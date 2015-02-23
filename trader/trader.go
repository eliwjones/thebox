package trader

import (
	"github.com/eliwjones/thebox/dispatcher"
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/interfaces"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
	"time"
)

type Trader struct {
	adapter    interfaces.Adapter                     // Adapter already connected to "Broker".
	adptrAct   chan bool                              // Used to signal activity between Trader/Adapter.
	allotments []int                                  // Placeholder .. not sure how will handle allotments.
	commission map[util.ContractType]map[string]int   // commission fees per type for base, unit.
	dispatcher *dispatcher.Dispatcher                 // My megaphone.
	multiplier map[util.ContractType]int              // Stocks trade in units of 1, Options in units of 100.
	orders     map[string]structs.Order               // Open (Closed?) orders.
	out        map[string]map[string]chan interface{} // Generally, Delta's heading out to dispatcher.
	PoIn       chan structs.ProtoOrder                // Generally, ProtoOrders coming in.
	positions  map[string]structs.Position            // Current outstanding positions.
}

func New(adapter interfaces.Adapter) *Trader {
	t := &Trader{}

	t.allotments = []int{300000}

	t.adapter = adapter
	t.adptrAct = make(chan bool, 1000)
	t.positions, _ = t.adapter.GetPositions()
	t.orders, _ = t.adapter.GetOrders("")

	t.PoIn = make(chan structs.ProtoOrder, 1000)
	t.out = make(map[string]map[string]chan interface{})

	t.multiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	t.commission = map[util.ContractType]map[string]int{}
	t.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	t.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	t.dispatcher = dispatcher.New(1000)

	// Grind ProtoOrders into Orders.
	go func() {
		for po := range t.PoIn {
			// Combine allotment with Path and send to Trader as ProtoOrder.
			o, err := t.constructOrder(po, t.allotments[0])
			if err != nil {
				if po.Reply != nil {
					po.Reply <- po
				}
				continue
			}

			// Submit order for execution.
			oid, err := t.submitOrder(o)
			if po.Reply != nil {
				if err != nil {
					po.Reply <- false
				} else {
					po.Reply <- oid
				}
			}
		}
	}()

	// Sync Orders, Positions and reap Deltas from t.adapter?
	go func() {
		for {
			select {
			case <-t.adptrAct:
				// Sleep 1 ms then Drain Channel.
				// This is used primarily for Simulation when actions are accelerated.
				time.Sleep(time.Duration(1) * time.Millisecond)
				for len(t.adptrAct) > 0 {
					<-t.adptrAct
				}
				t.sync()
			case <-time.After(time.Duration(30) * time.Second):
				t.sync()
			}
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

func (t *Trader) submitOrder(o structs.Order) (string, error) {
	id, err := t.adapter.SubmitOrder(o)
	t.adptrAct <- true
	return id, err
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
