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
		cash := 0
		maxcash := 0
		closed := 0
		commissions := 0
		volume := 0
		for _, p := range t.Histories {
			totalPositions += 1

			pcash := p.Volume * 100 * (p.LimitClose - p.Open)
			cash += pcash
			maxpcash := p.Volume * 100 * (p.MaxClose - p.Open)
			maxcash += maxpcash
			if p.Closed {
				closed += 1
				commissions += p.Commission
				pcash -= p.Commission
				maxpcash -= p.Commission
			}
			commissions += p.Commission
			volume += p.Volume

			// Collect position returns.
			pcash -= p.Commission
			maxpcash -= p.Commission

			positionReturns = append(positionReturns, float64(pcash)/300000)
			maxPositionReturns = append(maxPositionReturns, float64(maxpcash)/300000)
		}
		maxreturn := float64(maxcash-commissions) / 300000
		maxreturns = append(maxreturns, maxreturn)
		returns = append(returns, float64(cash-commissions)/300000)
		positionCounts = append(positionCounts, float64(t.PositionCount))

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
