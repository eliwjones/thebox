package destiny

import (
	"github.com/eliwjones/thebox/collector"
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"fmt"
	"math"
	"os"
	"sort"
)

type Destiny struct {
	collector      *collector.Collector
	dataDir        string                      // top level dir for data.
	edges          map[int64][]structs.Maximum // edges keyed by TimestampID().
	edgeMultiplier float64
	id             string // allows for namespacing and multiple simulation runs.
	PoC            chan structs.ProtoOrder
	Pulses         chan int64 // timestamps from pulsar come here.
	PulsarReply    chan int64 // Reply back to Pulsar when done doing work.
	underlying     string
	weeksBack      int
}

func New(id string, dataDir string, underlying string, weeksBack int, edgeMultiplier float64, c *collector.Collector, poc chan structs.ProtoOrder) *Destiny {
	d := &Destiny{id: id, dataDir: dataDir, edgeMultiplier: edgeMultiplier,
		underlying: underlying, weeksBack: weeksBack}
	d.collector = c
	d.edges = map[int64][]structs.Maximum{}
	d.PoC = poc
	d.Pulses = make(chan int64, 1000)
	d.PulsarReply = make(chan int64, 1000)

	go d.processPulses()

	return d
}

func (d *Destiny) processPulses() {
	currentWeekID := int64(0)
	for timestamp := range d.Pulses {
		if timestamp == -1 {
			// Serialize state in preparation for shutdown.
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

			edgePremiumPct := funcs.PremiumPct(edge.OptionAsk, edge.Strike, float64(2.2))
			nearestPremiumPct := float64(1)
			matchOption := structs.Option{}
			for _, quote := range quotes {
				if quote.Type != edge.OptionType {
					continue
				}
				premiumPct := funcs.PremiumPct(quote.Ask, quote.Strike, float64(2.2))
				if math.Abs(premiumPct-edgePremiumPct) < math.Abs(nearestPremiumPct-edgePremiumPct) {
					matchOption = quote
					nearestPremiumPct = premiumPct
				}

			}
			if matchOption.Symbol == "" {
				fmt.Printf("[%d] Empty matchOption, continuing.\n", timestamp)
				continue
			}

			// Determine if matchOption could return "close enough" multiplier.
			// This is completely naive method.
			// Could block if not close enough, OR could simply submit order with Min(edge.Ask, matchOption.Ask)

			edgeMultiplier := funcs.Multiplier(edge.MaximumBid, edge.OptionAsk, 2.2)
			matchOptionMultiplier := funcs.Multiplier(edge.MaximumBid, matchOption.Ask, 2.2)
			multiplierDiff := math.Abs(edgeMultiplier - matchOptionMultiplier)
			// Want multiplier to be within 10% of edge Multiplier.
			// If it is too far away, then I'm in uncharted territory that would require more thought.
			if multiplierDiff/edgeMultiplier > float64(0.1) {
				fmt.Printf("Multiplier Diff Too Big: %.4f\n", multiplierDiff)
				continue
			}

			// Seconds to Max
			secondsToMax := edge.MaxTimestamp - edge.Timestamp

			// Construct PO.
			po := structs.ProtoOrder{}
			po.Timestamp = timestamp
			po.Symbol = matchOption.Symbol
			po.LimitOpen = matchOption.Ask
			po.LimitTS = timestamp + secondsToMax
			po.Type = util.OPTION
			po.Underlying = d.underlying

			// Send to ProtoOrder Channel.
			d.PoC <- po
		}

		// Done doing my thing.  Send
		d.PulsarReply <- timestamp
	}
}

func (d *Destiny) populateEdges(timestamp int64) {
	edges := []structs.Maximum{}

	filename := fmt.Sprintf("%d", funcs.WeekID(timestamp))
	path := fmt.Sprintf("%s/destiny/%02d_week_edges", d.dataDir, d.weeksBack)
	edgeData, err := os.ReadFile(path + "/" + filename)
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
		edges, err = d.collector.DeserializeMaximums(string(edgeData))
		if err != nil {
			panic("Someone broke something with DeserializeMaximums or SerializeMaximums.")
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
	path = fmt.Sprintf("%s/%s/destiny/chosen_edges", d.dataDir, d.id)
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
