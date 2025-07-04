package trader

import (
	"github.com/eliwjones/thebox/collector"
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/interfaces"
	"github.com/eliwjones/thebox/util/structs"

	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"time"
)

type PostionHistory struct {
	Commission    int    // How much is commission to open trade (presumably would be same to close.)
	Closed        bool   // Did position successfully close?
	LimitClose    int    // Limit for closing position.
	MaxClose      int    // What does GetMax(timestamp, underlying, symbol) show was MaxBid.
	MaxTimestamp  int64  // When did MaxBid occur.
	Open          int    // Open price for position.
	OpenTimestamp int64  // When was position opened.
	Symbol        string // Option symbol.
	Timestamp     int64  // When was position closed.
	UltimateTS    int64  // Timestamp when all were finalized?
	Underlying    string // Underlying.. still funky that need this for querying collector.
	Volume        int    // How many.

	// Stuff return info here.
	TSdiff    int64
	Return    float64
	MaxReturn float64
}

type ByOpenTimestamp []PostionHistory

func (ph ByOpenTimestamp) GetReturns() []float64 {
	returns := []float64{}
	for _, p := range ph {
		returns = append(returns, p.Return)
	}
	return returns
}
func (ph ByOpenTimestamp) GetMaxReturns() []float64 {
	returns := []float64{}
	for _, p := range ph {
		returns = append(returns, p.MaxReturn)
	}
	return returns
}
func (ph ByOpenTimestamp) GetTradeTime() []float64 {
	returns := []float64{}
	for _, p := range ph {
		returns = append(returns, float64(p.OpenTimestamp-p.UltimateTS))
	}
	return returns
}
func (ph ByOpenTimestamp) GetTSdiff() []float64 {
	returns := []float64{}
	for _, p := range ph {
		returns = append(returns, float64(p.TSdiff))
	}
	return returns
}
func (ph ByOpenTimestamp) GetUltimateTS() []float64 {
	returns := []float64{}
	for _, p := range ph {
		returns = append(returns, float64(p.UltimateTS))
	}
	return returns
}
func (ph ByOpenTimestamp) Len() int      { return len(ph) }
func (ph ByOpenTimestamp) Swap(i, j int) { ph[i], ph[j] = ph[j], ph[i] }
func (ph ByOpenTimestamp) Less(i, j int) bool {
	return ph[i].OpenTimestamp < ph[j].OpenTimestamp
}

type Historae struct {
	Histories []PostionHistory

	// Holders for finalized data.
	Commission         int       // Total commission payout.
	MaxPositionReturns []float64 // Individual returns for each "max" position.  Compared to total account value.
	MaxReturn          float64   // Highest possible return.
	PositionCount      int       // Total number of positions opened.
	PositionReturns    []float64 // Individual returns for each position.  Compared to total account value.
	Return             float64   // Actual return.
	TSdiffs            []float64 // How far was position close from MaxTimestamp.
}

// Seems annoying that have to explain how get len([]slice) swap(i,j).
type ByMaxReturn []Historae

func (h ByMaxReturn) Len() int      { return len(h) }
func (h ByMaxReturn) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h ByMaxReturn) Less(i, j int) bool {
	return h[i].MaxReturn < h[j].MaxReturn
}

type ByReturn []Historae

func (h ByReturn) Len() int      { return len(h) }
func (h ByReturn) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h ByReturn) Less(i, j int) bool {
	return h[i].Return < h[j].Return
}

func (t *Trader) FinalizeHistorae(startCash int) {
	t.Historae = Historae{}
	t.Historae.Histories = []PostionHistory{}
	for _, p := range t.PositionHistory {
		t.Historae.Histories = append(t.Historae.Histories, p)
	}

	positionReturns := []float64{}
	maxPositionReturns := []float64{}
	tsdiffs := []float64{}

	cash := 0
	maxcash := 0
	closed := 0
	commissions := 0
	volume := 0
	positionCount := 0

	for idx, p := range t.Historae.Histories {
		positionCount += 1
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

		// TSdiff
		tsdiff := p.Timestamp - p.MaxTimestamp
		if p.MaxTimestamp == 0 {
			tsdiff = p.Timestamp - p.UltimateTS
		}
		t.Historae.Histories[idx].TSdiff = tsdiff

		preturn := float64(100*pcash) / float64(startCash)
		t.Historae.Histories[idx].Return = preturn

		maxpreturn := float64(100*maxpcash) / float64(startCash)
		t.Historae.Histories[idx].MaxReturn = maxpreturn

		positionReturns = append(positionReturns, preturn)
		maxPositionReturns = append(maxPositionReturns, maxpreturn)

		tsdiffs = append(tsdiffs, float64(tsdiff))
	}
	t.Historae.Commission = commissions
	t.Historae.Return = float64(100*(cash-commissions)) / float64(startCash)
	t.Historae.MaxReturn = float64(100*(maxcash-commissions)) / float64(startCash)
	t.Historae.PositionCount = positionCount

	t.Historae.PositionReturns = positionReturns
	t.Historae.MaxPositionReturns = maxPositionReturns

	t.Historae.TSdiffs = tsdiffs
}

type Tracker struct {
	Distance      int64 // How far apart should timestamps be?
	LastTimestamp int64 // Need to know when to look for max.
	LastSample    int64 // What is timestamp when we last sampled?
	RemainingTS   int   // How many left so can calculate backoff.
	SamplesNeeded int   // How many should we sample before seeking Max?
	Samples       []int // History of sampled Bids.
}

type Trader struct {
	adapter         interfaces.Adapter                   `json:"-"`                  // Adapter already connected to "Broker".
	Allotments      []int                                `json:"allotments"`         // Placeholder .. not sure how will handle allotments.
	Balances        map[string]int                       `json:"balances"`           // Not sure on wisdom of rolling Money into Trader, but we shall see.
	c               *collector.Collector                 `json:"-"`                  // For collector.GetQuote()
	commission      map[util.ContractType]map[string]int `json:"-"`                  // commission fees per type for base, unit.
	CurrentWeekId   int64                                `json:"currentWeekId"`      // When am I?
	dataDir         string                               `json:"-"`                  // Where am I?
	Historae        Historae                             `json:"historae,omitempty"` // Struct with all my History info.
	id              string                               `json:"-"`                  // Who am I?
	multiplier      map[util.ContractType]int            `json:"-"`                  // Stocks trade in units of 1, Options in units of 100.
	orders          map[string]structs.Order             `json:"-"`                  // Open (Closed?) orders.
	PoIn            chan structs.ProtoOrder              `json:"-"`                  // Generally, ProtoOrders coming in.
	Positions       map[string]structs.Position          `json:"positions"`          // Current outstanding positions.
	PositionCount   int                                  `json:"positioncount"`      // How many Positions have I opened?
	PositionHistory map[string]PostionHistory            // Information pertaining to open, close, commission, max.
	Pulses          chan int64                           `json:"-"`         // timestamps from pulsar come here.
	PulsarReply     chan int64                           `json:"-"`         // Reply back to Pulsar when done doing work.
	Trackers        map[string]Tracker                   `json:"trackers"`  // Sampled bids for currently open positions.  Used for Optimal Stopping.
	traderDir       string                               `json:"-"`         // Where to save information pertaining to this instance of trader.
	WeekCount       int                                  `json:"weekcount"` // Count weeks I have seen.
}

func New(id string, dataDir string, adapter interfaces.Adapter, c *collector.Collector) *Trader {
	t := &Trader{id: id, dataDir: dataDir}

	t.adapter = adapter
	t.c = c
	t.commission = adapter.Commission()
	t.multiplier = adapter.ContractMultiplier()
	t.PositionHistory = map[string]PostionHistory{}
	t.Positions = map[string]structs.Position{}
	t.orders, _ = t.adapter.GetOrders("")

	t.PoIn = make(chan structs.ProtoOrder, 1000)
	t.Pulses = make(chan int64, 1000)
	t.PulsarReply = make(chan int64, 1000)
	t.Trackers = map[string]Tracker{}
	t.traderDir = fmt.Sprintf("%s/%s/trader", t.dataDir, t.id)

	// Ambivalent about need for big, official SerializeAndSaveState() functions..
	serializedState, _ := os.ReadFile(t.traderDir + "/state")
	t.deserializeState(serializedState)

	// Sync may overwrite saved state since adapter is source of truth.
	t.sync(int64(-1))
	// If trade comes in on first timestamp.. need to already have Allotments initialized..
	t.Allotments = allotments(t.Balances["cash"], t.Balances["value"])

	// Sync Orders, Positions and reap Deltas from t.adapter?
	go func() {
		lastTimestamp := int64(0)
		for timestamp := range t.Pulses {
			weekID := funcs.WeekID(timestamp)
			if t.CurrentWeekId != weekID && timestamp != -1 {
				// Reset any open Positions as they have expired worthless.
				t.Positions = map[string]structs.Position{}
				t.adapter.Reset()

				t.WeekCount += 1

				// init or get allotments.
				t.Allotments = allotments(t.Balances["cash"], t.Balances["value"])
				// Anything to log if Tracker is non-empty?
				// Generally would mean at least one position expired worthless.
				t.Trackers = map[string]Tracker{}

				// Finalize Histories.
				for id, history := range t.PositionHistory {
					if history.UltimateTS != 0 {
						continue
					}
					history.UltimateTS = lastTimestamp
					maximum, err := t.c.GetMaximum(history.OpenTimestamp, history.Symbol)
					if err == nil {
						history.MaxClose = maximum.MaximumBid
						history.MaxTimestamp = maximum.MaxTimestamp
					}
					t.PositionHistory[id] = history
				}

				t.CurrentWeekId = weekID
			}
			t.consumePoIn(timestamp)

			if timestamp == -1 {
				// Save State.
				serializedState, _ := json.Marshal(t)
				funcs.LazyWriteFile(t.traderDir, "state", serializedState)

				t.PulsarReply <- timestamp
				return
			}

			t.sync(timestamp)

			for positionId, tracker := range t.Trackers {
				// Get quote for option symbol for current timestamp from collector.
				// Will need to fix collector.GetQuotes(underlying, timestamp) and add GetQuote(symbol, underlying, timestamp)
				p := t.Positions[positionId]
				q, err := t.c.GetQuote(timestamp, p.Order.ProtoOrder.Underlying, p.Order.Symbol)
				if err != nil {
					// This breaks tests for ProtoOrder submissions.. since I'm passing in invalid timestamp.
					// Comment out until can pass in kosher timestamps for testing.. or create cleaner test.
					//panic("What broke?")
					continue
				}
				stopv1 := t.optimalStopV1(timestamp, tracker, positionId, q)
				stopv2 := false // t.optimalStopV2(timestamp, tracker, positionId, q)

				if stopv1 || stopv2 {
					t.adapter.ClosePosition(positionId, q.Bid)

					// Add new info to Histories.
					history := t.PositionHistory[positionId]
					history.LimitClose = q.Bid
					history.Timestamp = timestamp
					t.PositionHistory[positionId] = history

					logLine := fmt.Sprintf("%d,order-close,%s,%d", timestamp, positionId, q.Bid)
					funcs.LazyAppendFile(t.traderDir, "log", logLine)
				}
			}
			lastTimestamp = timestamp
			t.PulsarReply <- timestamp
		}
	}()

	return t
}

func (t *Trader) constructOrder(po structs.ProtoOrder, allotment int) (structs.Order, error) {
	o := structs.Order{Symbol: po.Symbol, Type: po.Type}
	o.ProtoOrder = po
	o.Volume = (allotment - t.commission[o.Type]["base"]) / (po.LimitOpen * t.multiplier[o.Type])
	o.Limitprice = po.LimitOpen
	o.Maxcost = (o.Volume * o.Limitprice * t.multiplier[o.Type]) + (o.Volume * t.commission[o.Type]["unit"])
	// Lazy search for acceptable volume.
	for o.Maxcost > (allotment - t.commission[o.Type]["base"]) {
		o.Volume--
		o.Maxcost = (o.Volume * o.Limitprice * t.multiplier[o.Type]) + (o.Volume * t.commission[o.Type]["unit"])
	}
	if o.Volume <= 0 {
		return o, errors.New("impossible order. not enough allotment to cover commission")
	}
	return o, nil
}

func (t *Trader) consumePoIn(timestamp int64) {
	for len(t.PoIn) > 0 {
		po := <-t.PoIn
		allotment := 0
		if len(t.Allotments) > 0 {
			allotment, t.Allotments = t.Allotments[0], t.Allotments[1:]
		}
		o, err := t.constructOrder(po, allotment)
		if err != nil {
			if po.Reply != nil {
				po.Reply <- po
			}
			continue
		}

		// Submit order for execution.
		oid, err := t.adapter.SubmitOrder(o)
		// Log order submission.
		o.Id = oid
		encodedOrder, _ := funcs.Encode(&o, funcs.OrderEncodingOrder)
		if err != nil {
			encodedOrder = fmt.Sprintf("%d,error,%s,%s", timestamp, err, encodedOrder)
		} else {
			encodedOrder = fmt.Sprintf("%d,order,%s", timestamp, encodedOrder)
		}
		funcs.LazyAppendFile(t.traderDir, "log", encodedOrder)

		if po.Reply != nil {
			if err != nil {
				po.Reply <- false
			} else {
				po.Reply <- oid
			}
		}
	}
}

func (t *Trader) deserializeState(state []byte) error {
	// Load allotments, currentWeekId, positions
	dt := &Trader{}
	err := json.Unmarshal(state, &dt)

	if err != nil {
		return err
	}
	t.Allotments = dt.Allotments
	t.Balances = dt.Balances
	t.CurrentWeekId = dt.CurrentWeekId
	t.Historae = dt.Historae
	t.Positions = dt.Positions
	t.PositionCount = dt.PositionCount
	t.Trackers = dt.Trackers
	t.WeekCount = dt.WeekCount

	return nil
}

func (t *Trader) initHistory(p structs.Position, timestamp int64) {
	history := PostionHistory{}
	history.Commission = p.Commission
	history.Open = p.Fillprice
	history.OpenTimestamp = timestamp
	history.Symbol = p.Order.Symbol
	history.Underlying = p.Order.ProtoOrder.Underlying
	history.Volume = p.Order.Volume

	t.PositionHistory[p.Id] = history
}

func (t *Trader) initTracking(p structs.Position, timestamp int64) {
	// Fake it by estimating.
	// 40 per day. (assuming 10 min interval.)
	distance := int(time.Friday) - int(time.Unix(timestamp, 0).Weekday())
	timestamps := distance * 40

	// Plus however many seconds it is until 16:00:00 "today".
	// Again, all assuming 10 minute intervals.
	hour, min, _ := time.Unix(timestamp, 0).Clock()
	if hour < 16 {
		timestamps += (16-(hour+1))*6 + (60-min)/10
	}

	timestamps -= 2             // Feels like this is necessary.
	timestamps = timestamps / 5 // Really just want 50 minute intervals.
	interval := int64(50 * 60)  // 50 minutes in seconds.

	tracker := Tracker{Distance: interval, RemainingTS: timestamps}
	tracker.SamplesNeeded = int(float64(timestamps) / math.Exp(1))
	tracker.SamplesNeeded = int(float64(timestamps) / 1.5)

	tracker.Samples = []int{p.Fillprice}
	tracker.LastSample = timestamp
	tracker.LastTimestamp = timestamp
	_, exists := t.Trackers[p.Id]
	if exists {
		panic("Someone fucked something up. Either am getting duplicate Order/Position IDs or a Position has disappeared and re-appeared.")
	}
	t.Trackers[p.Id] = tracker
}

func (t *Trader) optimalStopV1(timestamp int64, tracker Tracker, positionId string, q structs.Option) bool {
	if timestamp-tracker.LastTimestamp < tracker.Distance {
		// Not enough time has passed, so move along.
		return false
	}
	tracker.LastTimestamp = timestamp
	tracker.RemainingTS -= 1

	//p := t.Positions[positionId]

	if tracker.SamplesNeeded <= 0 { // || timestamp > p.Order.ProtoOrder.LimitTS {
		// Check if current bid is greater than max(tracker.Samples)
		isMax := true

		// Poor Man's Backoff.
		canBeLessThan := len(tracker.Samples) - tracker.RemainingTS/2

		for _, bid := range tracker.Samples {
			if bid >= q.Bid {
				if canBeLessThan > 0 {
					canBeLessThan -= 1
					continue
				}
				// Can't be max since sampled item is bigger.
				// Could use a labeled break, but feels wrong.
				isMax = false
				break
			}
		}
		t.Trackers[positionId] = tracker
		if isMax {
			return true
		}
	}
	if tracker.SamplesNeeded > 0 {
		tracker.Samples = append(tracker.Samples, q.Bid)

		tracker.SamplesNeeded -= 1 // Not correcting for holidays or gaps, so be warned.
		tracker.LastSample = timestamp

		t.Trackers[positionId] = tracker
	}
	return false
}

func (t *Trader) optimalStopV2(timestamp int64, tracker Tracker, positionId string, q structs.Option) bool {
	if (q.Bid / t.Positions[positionId].Fillprice) >= 5 {
		return true
	}
	return false
}

func (t *Trader) sync(timestamp int64) {
	b, err := t.adapter.GetBalances()
	if err == nil {
		t.Balances = b
	}

	// Reconcile Orders, Positions.
	currentorders, err := t.adapter.GetOrders("")
	if err == nil {
		t.orders = currentorders
	}
	currentpositions, err := t.adapter.GetPositions()
	if err == nil {
		// Add new positions.
		for id, p := range currentpositions {
			_, found := t.Positions[id]
			if found {
				continue
			}
			// This is a new position.
			// Initialize tracker with counter so can start watching Bids.
			fmt.Printf("\n[Trader] New Positions: %v\n", p)
			t.initTracking(p, timestamp)
			t.initHistory(p, timestamp)
			t.Positions[id] = p
			t.PositionCount += 1
		}
		// Delete old positions and trackers
		for id := range t.Positions {
			_, found := currentpositions[id]
			if found {
				continue
			}
			delete(t.Positions, id)
			delete(t.Trackers, id)

			// Is it really necessary to always do this dance with a map[string]struct{} ?
			history := t.PositionHistory[id]
			history.Closed = true
			t.PositionHistory[id] = history
		}
	}
}

func allotments(cash int, value int) []int {
	a := cash / 100
	// 10 1% allotments
	allotments := []int{}
	count := min(cash/a, 10)
	for range count {
		allotments = append(allotments, a)
	}
	return allotments
}
