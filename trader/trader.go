package trader

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/interfaces"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
)

type Trader struct {
	adapter       interfaces.Adapter                   // Adapter already connected to "Broker".
	allotments    []int                                // Placeholder .. not sure how will handle allotments.
	commission    map[util.ContractType]map[string]int // commission fees per type for base, unit.
	currentWeekId int64
	multiplier    map[util.ContractType]int   // Stocks trade in units of 1, Options in units of 100.
	orders        map[string]structs.Order    // Open (Closed?) orders.
	PoIn          chan structs.ProtoOrder     // Generally, ProtoOrders coming in.
	positions     map[string]structs.Position // Current outstanding positions.
	Pulses        chan int64                  // timestamps from pulsar come here.
	PulsarReply   chan int64                  // Reply back to Pulsar when done doing work.
}

func New(adapter interfaces.Adapter) *Trader {
	t := &Trader{}

	t.adapter = adapter
	t.positions, _ = t.adapter.GetPositions()
	t.orders, _ = t.adapter.GetOrders("")

	t.PoIn = make(chan structs.ProtoOrder, 1000)
	t.Pulses = make(chan int64, 1000)
	t.PulsarReply = make(chan int64, 1000)

	t.multiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	t.commission = map[util.ContractType]map[string]int{}
	t.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	t.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	// Sync Orders, Positions and reap Deltas from t.adapter?
	go func() {
		for timestamp := range t.Pulses {
			t.consumePoIn(timestamp)

			if timestamp == -1 {
				// Serialize state.

				t.PulsarReply <- timestamp
				return
			}
			weekID := funcs.WeekID(timestamp)
			if t.currentWeekId != weekID {
				// init or get allotments.
				t.allotments = allotments()

				t.currentWeekId = weekID
			}

			t.sync()

			t.PulsarReply <- timestamp
		}
	}()

	return t
}

func (t *Trader) constructOrder(po structs.ProtoOrder, allotment int) (structs.Order, error) {
	o := structs.Order{Symbol: po.Symbol, Type: po.Type}
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
		o, err := t.constructOrder(po, t.allotments[0])
		if err != nil {
			if po.Reply != nil {
				po.Reply <- po
			}
			continue
		}

		// Submit order for execution.
		oid, err := t.adapter.SubmitOrder(o)

		if po.Reply != nil {
			if err != nil {
				po.Reply <- false
			} else {
				po.Reply <- oid
			}
		}
	}
}

func (t *Trader) deserializeState() {
	// Load allotments, currentWeekId, positions
}

func (t *Trader) serializeState() {
	// What gets saved here?
	// For cron-ed, single timestamp behaviour, presumably need current:
	//     allotments, currentWeekId, positions
	//  Positions need be serialized so can have associated 'order-id' and sampled bids and corresponding sample periods.
}

func (t *Trader) sync() {
	// Reconcile Orders, Positions.
	currentorders, err := t.adapter.GetOrders("")
	if err != nil {
		t.orders = currentorders
	}
	currentpositions, err := t.adapter.GetPositions()
	if err != nil {
		for id, _ := range t.positions {
			_, found := currentpositions[id]
			if !found {
				// If position no longer found, must calculate Delta.
				// Presumably link order-id to position and then back to closing order.
			}
		}

		t.positions = currentpositions
	}
}

func allotments() []int {
	// 15 $3,000 allotments
	allotments := []int{}
	for i := 0; i < 15; i++ {
		allotments = append(allotments, 3000*100)
	}
	return allotments
}
