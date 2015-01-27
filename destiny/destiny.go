package destiny

import (
	"github.com/eliwjones/thebox/collector"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"sort"
)

type Destiny struct {
	collector      *collector.Collector
	dataDir        string                      // top level dir for data.
	edges          map[int64][]structs.Maximum // edges keyed by TimestampID().
	edgeMultiplier float64
	id             string     // allows for namespacing and multiple simulation runs.
	Pulses         chan int64 // timestamps from pulsar come here.
	PulsarReply    chan int64 // Reply back to Pulsar when done doing work.
	underlying     string
	weeksBack      int
}

func New(c *collector.Collector, dataDir string, edgeMultiplier float64, id string, underlying string, weeksBack int) *Destiny {
	d := &Destiny{id: id, dataDir: dataDir, edgeMultiplier: edgeMultiplier,
		underlying: underlying, weeksBack: weeksBack}
	d.collector = c
	d.edges = map[int64][]structs.Maximum{}
	d.Pulses = make(chan int64, 1000)
	d.PulsarReply = make(chan int64, 1000)

	go d.processPulses()

	return d
}

func (d *Destiny) processPulses() {
	currentWeekID := int64(0)
	for timestamp := range d.Pulses {
		if timestamp == -1 {
			d.PulsarReply <- timestamp
			return
		}
		// if Change week, load new edges.
		weekID := funcs.WeekID(timestamp)
		if currentWeekID != weekID {
			d.populateEdges(timestamp)
			currentWeekID = weekID
		}

		quotes, _ := d.collector.GetQuotes(timestamp, d.underlying)

		//Grind into ProtoOrders to send to Trader.
		for _, edge := range d.edges[funcs.TimestampID(timestamp)] {
			// Grind into Order.  Send to Trader.
			// Find quote nearest to edge.
			edgePremiumPct := (float64(edge.OptionAsk) + float64(2.2)) / float64(edge.Strike)
			nearestPremiumPct := float64(1)
			matchOption := structs.Option{}
			for _, quote := range quotes {
				if quote.Type != edge.OptionType {
					continue
				}
				premiumPct := (float64(quote.Ask) + float64(2.2)) / float64(quote.Strike)
				if math.Abs(premiumPct-edgePremiumPct) < math.Abs(nearestPremiumPct-edgePremiumPct) {
					matchOption = quote
					nearestPremiumPct = premiumPct
				} else {
					fmt.Printf("%.4f > %.4f\n", premiumPct, edgePremiumPct)
				}

			}
			if matchOption.Symbol == "" {
				fmt.Printf("[%d] Empty matchOption, continuing.\n", timestamp)
				continue
			}

			// Determine if matchOption could return "close enough" multiplier.
			// This is completely naive method.
			// Could block if not close enough, OR could simply submit order with Min(edge.Ask, matchOption.Ask)

			encodedTrade := fmt.Sprintf("%d,%s,%d,%d", timestamp, matchOption.Symbol, matchOption.Ask, edge.MaximumBid)

			edgeMultiplier := float64(edge.MaximumBid) / (float64(edge.OptionAsk) + float64(2.2))
			matchOptionMultiplier := float64(edge.MaximumBid) / (float64(matchOption.Ask) + float64(2.2))
			multiplierDiff := math.Abs(edgeMultiplier - matchOptionMultiplier)
			// Want multiplier to be within 10% of edge Multiplier.
			// If it is too far away, then I'm in uncharted territory that would require more thought.
			if multiplierDiff/edgeMultiplier > float64(0.1) {
				fmt.Printf("Multiplier Diff Too Big: %d\n", multiplierDiff)
				fmt.Println(encodedTrade)
				continue
			}

			// Presumably, have "kosher" matchOption. Construct Open and Close Orders.
			// "Easy" examination would be... save "order" for:
			//     Timestamp, option.Symbol, LimitOpen == matchOption.Ask, LimitClose == edge.MaximumBid.
			// Can then compare to Maximum for Timestamp, option.Symbol at end of week calculation.
			filename := fmt.Sprintf("%d", weekID)
			path := fmt.Sprintf("%s/%s/trades", d.dataDir, d.id)

			err := funcs.LazyAppendFile(path, filename, encodedTrade)
			if err != nil {
				fmt.Println(err)
			}
		}

		// Done doing my thing.  Send
		d.PulsarReply <- timestamp
	}
}

func (d *Destiny) populateEdges(timestamp int64) {
	edges := []structs.Maximum{}

	filename := fmt.Sprintf("%d", funcs.WeekID(timestamp))
	path := fmt.Sprintf("%s/edges/%d", d.dataDir, d.weeksBack)
	edgeData, err := ioutil.ReadFile(path + "/" + filename)
	if err != nil {
		// Not found, load from collector and persist.
		fmt.Printf("[populateEdges] %s\n", err)
		edges = d.collector.GetPastNEdges(timestamp, d.weeksBack)

		// Encode and save for future reference.
		encodedEdges, _ := d.collector.SerializeMaximums(edges)
		err = funcs.LazyWriteFile(path, filename, []byte(encodedEdges))
		if err != nil {
			message := fmt.Sprintf("Failed to write encodedEdges. Err: %s", err)
			panic(message)
		}
	} else {
		// Got edgeData, decode into []structs.Maximum
		for _, edge := range bytes.Split(edgeData, []byte("\n")) {
			e := structs.Maximum{}
			err := funcs.Decode(string(edge), &e, funcs.MaximumEncodingOrder)
			if err != nil {
				panic("What to do?  Thought I had edges, but they buggy.  Just get from collector?")
			}
			edges = append(edges, e)
		}
	}

	// Filter out unwanted symbols.
	edges = filterEdgesByUnderlying(edges, d.underlying)

	// Choose 30 Edges.
	bag := funcs.ChooseMFromN(30, len(edges))
	d.edges = map[int64][]structs.Maximum{}
	for _, index := range bag {
		edge := edges[index]
		timestampID := funcs.TimestampID(edge.Timestamp)
		d.edges[timestampID] = append(d.edges[timestampID], edge)
	}
	// Limit to multipliers of interest. Also, has effect of removing gaps.
	for timestampID, _ := range d.edges {
		d.edges[timestampID] = filterEdgesByMultiplier(d.edges[timestampID], d.edgeMultiplier)
	}

	// Save d.edges to disk so can compare to actual constructed "orders"?
	toBeSerialized := []structs.Maximum{}
	for _, edges := range d.edges {
		toBeSerialized = append(toBeSerialized, edges...)
	}
	sort.Sort(collector.ByTimestampID(toBeSerialized))
	encodedEdges, _ := d.collector.SerializeMaximums(toBeSerialized)
	path = fmt.Sprintf("%s/%s/chosen_edges/%d", d.dataDir, d.id, d.weeksBack)
	funcs.LazyWriteFile(path, filename, []byte(encodedEdges))
}

func filterEdgesByMultiplier(edges []structs.Maximum, multiplier float64) []structs.Maximum {
	filteredEdges := []structs.Maximum{}
	for _, edge := range edges {
		edgeMultiple := float64(edge.MaximumBid) / (float64(edge.OptionAsk) + float64(2.2))
		if edgeMultiple < multiplier {
			continue
		}
		filteredEdges = append(filteredEdges, edge)
	}
	return filteredEdges
}

func filterEdgesByUnderlying(edges []structs.Maximum, underlying string) []structs.Maximum {
	filteredEdges := []structs.Maximum{}
	for _, edge := range edges {
		if edge.Underlying != underlying {
			continue
		}
		filteredEdges = append(filteredEdges, edge)
	}
	return filteredEdges
}
