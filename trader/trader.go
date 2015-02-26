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
)

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
	traderDir     string                               `json:"-"`             // Where to save information pertaining to this instance of trader.
}

func New(id string, dataDir string, adapter interfaces.Adapter) *Trader {
	t := &Trader{id: id, dataDir: dataDir}

	t.adapter = adapter
	t.Positions, _ = t.adapter.GetPositions()
	t.orders, _ = t.adapter.GetOrders("")

	t.PoIn = make(chan structs.ProtoOrder, 1000)
	t.Pulses = make(chan int64, 1000)
	t.PulsarReply = make(chan int64, 1000)
	t.traderDir = fmt.Sprintf("%s/%s/trader", t.dataDir, t.id)

	t.multiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	t.commission = map[util.ContractType]map[string]int{}
	t.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	t.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	// Ambivalent about need for big, official SerializeAndSaveState() functions..
	serializedState, _ := ioutil.ReadFile(t.traderDir + "/state")
	t.deserializeState(serializedState)

	// Sync may overwrite saved state since adapter is source of truth.
	t.sync()

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

				t.CurrentWeekId = weekID
			}

			t.sync()

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

	return nil
}

func (t *Trader) sync() {
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
		for id, _ := range t.Positions {
			_, found := currentpositions[id]
			if !found {
				// If position no longer found, must calculate Delta.
				// Presumably link order-id to position and then back to closing order.
			}
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
