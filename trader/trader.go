package trader

import (
	"github.com/eliwjones/thebox/dispatcher"
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/interfaces"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
)

type Trader struct {
	adapter     interfaces.Adapter                     // Adapter already connected to "Broker".
	adptrAct    chan bool                              // Used to signal activity between Trader/Adapter.
	allotments  []int                                  // Placeholder .. not sure how will handle allotments.
	commission  map[util.ContractType]map[string]int   // commission fees per type for base, unit.
	dispatcher  *dispatcher.Dispatcher                 // My megaphone.
	multiplier  map[util.ContractType]int              // Stocks trade in units of 1, Options in units of 100.
	orders      map[string]structs.Order               // Open (Closed?) orders.
	out         map[string]map[string]chan interface{} // Generally, Delta's heading out to dispatcher.
	PoIn        chan structs.ProtoOrder                // Generally, ProtoOrders coming in.
	positions   map[string]structs.Position            // Current outstanding positions.
	Pulses      chan int64                             // timestamps from pulsar come here.
	PulsarReply chan int64                             // Reply back to Pulsar when done doing work.
}

func New(adapter interfaces.Adapter) *Trader {
	t := &Trader{}

	t.allotments = []int{300000}

	t.adapter = adapter
	t.adptrAct = make(chan bool, 1000)
	t.positions, _ = t.adapter.GetPositions()
	t.orders, _ = t.adapter.GetOrders("")

	t.PoIn = make(chan structs.ProtoOrder, 1000)
	t.Pulses = make(chan int64, 1000)
	t.PulsarReply = make(chan int64, 1000)
	t.out = make(map[string]map[string]chan interface{})

	t.multiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	t.commission = map[util.ContractType]map[string]int{}
	t.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	t.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	t.dispatcher = dispatcher.New(1000)

	// Sync Orders, Positions and reap Deltas from t.adapter?
	go func() {
		for timestamp := range t.Pulses {
			t.consumePoIn(timestamp)

			if timestamp == -1 {
				// Must figure out if all t.PoIn messages have been consumed before shutting down.
				// Serialize state.
				t.PulsarReply <- timestamp
				return
			}
			// weekID := funcs.WeekID(timestamp)

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
		var oid string
		if timestamp > po.Timestamp {
			// Submit order for execution.
			oid, err = t.adapter.SubmitOrder(o)
		} else {
			// Save order to Order collection? (since orders can have future date?)
		}

		if po.Reply != nil {
			if err != nil {
				po.Reply <- false
			} else {
				po.Reply <- oid
			}
		}
	}
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
