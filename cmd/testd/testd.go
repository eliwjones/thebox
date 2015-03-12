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
	loops         = 100
	underlying    = "GOOG"
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
		t.Historae.Finalize(300000 * 100)

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

	fmt.Printf("Underlying: %s, WeekCount: %d, TotalPositions:%d\n", underlying, weekCount, totalPositions)

	max, min, med, avg := getDistribution(returns)
	fmt.Printf("Returns\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	max, min, med, avg = getDistribution(maxreturns)
	fmt.Printf("Maxreturns\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	max, min, med, avg = getDistribution(positionReturns)
	fmt.Printf("PositionReturns\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	max, min, med, avg = getDistribution(maxPositionReturns)
	fmt.Printf("MaxPositionReturns\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	max, min, med, avg = getDistribution(positionCounts)
	fmt.Printf("Positions\nMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)

	sort.Sort(trader.ByMaxReturn(historae))
	for idx, h := range historae {
		max, min, med, avg = getDistribution(h.MaxPositionReturns)
		fmt.Printf("MaxPositionReturns for %d-th Historae\n\tMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", idx, max, min, med, avg)
		fmt.Println(h.MaxPositionReturns)
		max, min, med, avg = getDistribution(h.PositionReturns)
		fmt.Printf("\tMax: %.2f, Min: %.2f, Med: %.2f, Avg: %.2f\n", max, min, med, avg)
		fmt.Println(h.PositionReturns)
	}

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
