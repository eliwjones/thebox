package destiny

import (
	"github.com/eliwjones/thebox/collector"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"bytes"
	"fmt"
	"io/ioutil"
)

type Destiny struct {
	collector      *collector.Collector
	dataDir        string                      // top level dir for data.
	edges          map[int64][]structs.Maximum // edges keyed by TimestampID().
	edgeMultiplier int
	id             string     // allows for namespacing and multiple simulation runs.
	Pulses         chan int64 // timestamps from pulsar come here.
	PulsarReply    chan int64 // Reply back to Pulsar when done doing work.
	underlying     string
	weeksBack      int
}

func New(collector *collector.Collector, dataDir string, edgeMultiplier int, id string, underlying string, weeksBack int) *Destiny {
	d := &Destiny{id: id, dataDir: dataDir, edgeMultiplier: edgeMultiplier,
		underlying: underlying, weeksBack: weeksBack}
	d.edges = map[int64][]structs.Maximum{}
	d.Pulses = make(chan int64, 1000)

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

		//Grind into ProtoOrders to send to Trader.
		for _, edge := range d.edges[funcs.TimestampID(timestamp)] {
			// Grind into Order.  Send to Trader.
			fmt.Println(edge)
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
		edges = d.collector.GetPastNEdges(timestamp, d.weeksBack)

		// Encode and save for future reference.
		encodedEdges := ""
		for _, edge := range edges {
			e, err := funcs.Encode(&edge, funcs.MaximumEncodingOrder)
			if err != nil {
				fmt.Println("[populateEdges] Err encoding edge: %s", err)
				continue
			}
			encodedEdges += e + "\n"
		}
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
}

func filterEdgesByMultiplier(edges []structs.Maximum, multiplier int) []structs.Maximum {
	filteredEdges := []structs.Maximum{}
	for _, edge := range edges {
		edgeMultiple := float64(edge.MaximumBid) / (float64(edge.OptionAsk) + float64(2.2))
		if edgeMultiple < float64(multiplier) {
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
