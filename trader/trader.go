package trader

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/interfaces"
	"github.com/eliwjones/thebox/util/structs"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"time"
)

type Tracker struct {
	Distance      int64 // How far apart should timestamps be?
	LastSample    int64 // What is timestamp when we last sampled?
	SamplesNeeded int   // How many should we sample before seeking Max?
	Samples       []int // History of sampled Bids.
}

type Trader struct {
	adapter       interfaces.Adapter                   `json:"-"`             // Adapter already connected to "Broker".
	Allotments    []int                                `json:"allotments"`    // Placeholder .. not sure how will handle allotments.
	Balances      map[string]int                       `json:"balances"`      // Not sure on wisdom of rolling Money into Trader, but we shall see.
	commission    map[util.ContractType]map[string]int `json:"-"`             // commission fees per type for base, unit.
	CurrentWeekId int64                                `json:"currentWeekId"` // When am I?
	dataDir       string                               `json:"-"`             // Where am I?
	id            string                               `json:"-"`             // Who am I?
	multiplier    map[util.ContractType]int            `json:"-"`             // Stocks trade in units of 1, Options in units of 100.
	orders        map[string]structs.Order             `json:"-"`             // Open (Closed?) orders.
	PoIn          chan structs.ProtoOrder              `json:"-"`             // Generally, ProtoOrders coming in.
	Positions     map[string]structs.Position          `json:"positions"`     // Current outstanding positions.
	Pulses        chan int64                           `json:"-"`             // timestamps from pulsar come here.
	PulsarReply   chan int64                           `json:"-"`             // Reply back to Pulsar when done doing work.
	Trackers      map[string]Tracker                   `json:"trackers"`      // Sampled bids for currently open positions.  Used for Optimal Stopping.
	traderDir     string                               `json:"-"`             // Where to save information pertaining to this instance of trader.
}

func New(id string, dataDir string, adapter interfaces.Adapter) *Trader {
	t := &Trader{id: id, dataDir: dataDir}

	t.adapter = adapter
	t.commission = adapter.Commission()
	t.multiplier = adapter.ContractMultiplier()
	t.Positions, _ = t.adapter.GetPositions()
	t.orders, _ = t.adapter.GetOrders("")

	t.PoIn = make(chan structs.ProtoOrder, 1000)
	t.Pulses = make(chan int64, 1000)
	t.PulsarReply = make(chan int64, 1000)
	t.Trackers = map[string]Tracker{}
	t.traderDir = fmt.Sprintf("%s/%s/trader", t.dataDir, t.id)

	// Ambivalent about need for big, official SerializeAndSaveState() functions..
	serializedState, _ := ioutil.ReadFile(t.traderDir + "/state")
	t.deserializeState(serializedState)

	// Sync may overwrite saved state since adapter is source of truth.
	t.sync(int64(-1))

	// Sync Orders, Positions and reap Deltas from t.adapter?
	go func() {
		for timestamp := range t.Pulses {
			t.consumePoIn(timestamp)

			if timestamp == -1 {
				// Save State.
				serializedState, _ := json.Marshal(t)
				funcs.LazyWriteFile(t.traderDir, "state", serializedState)

				t.PulsarReply <- timestamp
				return
			}
			weekID := funcs.WeekID(timestamp)
			if t.CurrentWeekId != weekID {
				// init or get allotments.
				t.Allotments = allotments(t.Balances["cash"], t.Balances["value"])
				// Anything to log if Tracker is non-empty?
				// Generally would mean at least one position expired worthless.
				t.Trackers = map[string]Tracker{}

				t.CurrentWeekId = weekID
			}

			t.sync(timestamp)

			// Sample Bids for Trackers.
			for positionId, tracker := range t.Trackers {
				// Get quote for option symbol for current timestamp from collector.
				// Will need to fix collector.GetQuotes(underlying, timestamp) and add GetQuote(symbol, underlying, timestamp)
				q := structs.Option{}

				if tracker.SamplesNeeded > 0 {
					if timestamp-tracker.LastSample < tracker.Distance {
						// Not enough time has passed, so move along.
						continue
					}
					tracker.Samples = append(tracker.Samples, q.Bid)

					tracker.SamplesNeeded -= 1 // Not correcting for holidays or gaps, so be warned.
					tracker.LastSample = timestamp

					t.Trackers[positionId] = tracker
				} else {
					// Check if current bid is greater than max(tracker.Samples)
					isMax := true
					for _, bid := range tracker.Samples {
						if bid > q.Bid {
							// Can't be max since sampled item is bigger.
							// Could use a labeled break, but feels wrong.
							isMax = false
							break
						}
					}
					if isMax {
						t.adapter.ClosePosition(positionId, q.Bid)
					}
				}
			}

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
	if o.Volume == 0 {
		return o, errors.New("Impossible order. Not enough Allotment to cover commission.")
	}
	return o, nil
}

func (t *Trader) consumePoIn(timestamp int64) {
	for len(t.PoIn) > 0 {
		po := <-t.PoIn
		o, err := t.constructOrder(po, t.Allotments[0])
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
	t.Positions = dt.Positions
	t.Trackers = dt.Trackers

	return nil
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

	timestamps -= 1             // Feels like this is necessary.
	timestamps = timestamps / 5 // Really just want 50 minute intervals.
	interval := int64(50 * 60)  // 50 minutes in seconds.

	tracker := Tracker{Distance: interval}
	tracker.SamplesNeeded = int(float64(timestamps) / math.Exp(1))
	tracker.Samples = []int{}
	_, exists := t.Trackers[p.Id]
	if exists {
		panic("Someone fucked something up. Either am getting duplicate Order/Position IDs or a Position has disappeared and re-appeared.")
	}
	t.Trackers[p.Id] = tracker
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
		for id, _ := range currentpositions {
			p, found := t.Positions[id]
			if found {
				continue
			}
			// This is a new position.
			// Initialize tracker with counter so can start watching Bids.
			t.initTracking(p, timestamp)
		}

		t.Positions = currentpositions
	}
}

func allotments(cash int, value int) []int {
	a := cash / 100
	// 15 1% allotments
	allotments := []int{}
	for i := 0; i < 15; i++ {
		allotments = append(allotments, a)
	}
	return allotments
}
