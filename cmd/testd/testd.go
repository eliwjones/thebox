package main

import (
	"github.com/eliwjones/thebox/adapter/simulate"
	"github.com/eliwjones/thebox/collector"
	"github.com/eliwjones/thebox/destiny"
	"github.com/eliwjones/thebox/pulsar"
	"github.com/eliwjones/thebox/trader"
	"github.com/eliwjones/thebox/util/funcs"

	"fmt"
	"runtime"
	"sort"
)

var (
	collectorRoot = "/home/mrz/go/src/github.com/eliwjones/thebox/cmd/collectord/testdir"
	c             = collector.New("test_collector", collectorRoot, 60)
	startTS       = ""
	stopTS        = ""
	loops         = 500
	underlying    = "AAPL"
	weeksBack     = 8
	multiplier    = 1.0
	realTime      = false
)

func main() {
	runtime.GOMAXPROCS(6)

	traderChannel := make(chan *trader.Trader, 1000)
	returns := []float64{}
	maxreturns := []float64{}
	positionReturns := []float64{}
	maxPositionReturns := []float64{}
	positionCounts := []float64{}
	totalPositions := 0
	//timestampDeltas := []float64{}

	historae := []trader.Historae{}

	// Jan 20 - 23rd
	//startTS = "1421764000"
	//stopTS = "1422046900"
	// Jan 26 - Jan 30
	//startTS = "1422288000"
	//stopTS = "1422651600"

	// Jan 26 - Mar 2nd
	startTS = "1422288000"
	stopTS = "1425311400"

	// Cheat to initialize edge data.
	t := runOnce()
	weekCount := t.WeekCount
	traderChannel <- t

	for i := 0; i < loops-1; i++ {
		go func() {
			t := runOnce()
			traderChannel <- t
		}()
	}

	resultsCounter := 0
	for t := range traderChannel {
		t.FinalizeHistorae(300000 * 100)

		positionReturns = append(positionReturns, t.Historae.PositionReturns...)
		maxPositionReturns = append(maxPositionReturns, t.Historae.MaxPositionReturns...)

		maxreturns = append(maxreturns, t.Historae.MaxReturn)
		returns = append(returns, t.Historae.Return)

		positionCounts = append(positionCounts, float64(t.Historae.PositionCount))

		historae = append(historae, t.Historae)

		resultsCounter += 1
		if resultsCounter == loops {
			break
		}
	}
	max, min, med, avg := float64(0), float64(0), float64(0), float64(0)
	sort.Sort(trader.ByMaxReturn(historae))
	for idx, h := range historae {
		fmt.Printf("%d-th Historae\n", idx)

		data := trader.ByOpenTimestamp(h.Histories)
		sort.Sort(data)

		printFloats(data.GetMaxReturns())
		printFloats(data.GetReturns())
		printTSDiffs(data.GetTSdiff())
		printTSDiffs(data.GetTradeTime())

		max, min, med, avg = getDistribution(h.MaxPositionReturns)
		fmt.Printf("\tMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)
		max, min, med, avg = getDistribution(h.PositionReturns)
		fmt.Printf("\tMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)
	}
	fmt.Printf("Underlying: %s, WeekCount: %d, TotalPositions:%d\n", underlying, weekCount, totalPositions)

	max, min, med, avg = getDistribution(returns)
	fmt.Printf("Returns\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	max, min, med, avg = getDistribution(maxreturns)
	fmt.Printf("Maxreturns\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	max, min, med, avg = getDistribution(positionReturns)
	fmt.Printf("PositionReturns\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	max, min, med, avg = getDistribution(maxPositionReturns)
	fmt.Printf("MaxPositionReturns\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	max, min, med, avg = getDistribution(positionCounts)
	fmt.Printf("Positions\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)
}

func getDistribution(slice []float64) (max float64, min float64, med float64, avg float64) {
	sort.Float64s(slice)
	min = slice[0]
	max = slice[len(slice)-1]
	med = slice[len(slice)/2]
	total := float64(0)
	for _, ct := range slice {
		total += ct
	}
	avg = total / float64(len(slice))

	return max, min, med, avg
}

func runOnce() *trader.Trader {
	p := pulsar.New(collectorRoot+"/live/timestamp", startTS, stopTS, true)

	id := funcs.ID(underlying, weeksBack, multiplier, realTime)
	a := simulate.New("simulate", "simulation", 300000*100)
	t := trader.New(id, "testDir", a, c)

	d := destiny.New(id, "testDir", underlying, weeksBack, multiplier, c, t.PoIn)

	p.Subscribe("destiny", d.Pulses, d.PulsarReply)
	p.Subscribe("trader", t.Pulses, t.PulsarReply)
	p.Start()

	return t
}

func printFloats(floats []float64) {
	fmt.Printf("[")

	for idx, val := range floats {
		if idx != 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%.2f", val)
	}

	fmt.Printf("]\n")
}

func printTSDiffs(tsdiffs []float64) {
	fmt.Printf("[")
	for idx, val := range tsdiffs {
		if idx != 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%d", int(val/(60.0*60.0)))
	}

	fmt.Printf("]\n")

}
