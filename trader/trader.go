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
	positions  map[string]structs.Position            // Current outstanding positions.
	orders     map[string]structs.Order               // Open (Closed?) orders.
	pomIn      chan structs.ProtoOrderMessage         // Generally, ProtoOrders coming in.
	out        map[string]map[string]chan interface{} // Generally, Delta's heading out to dispatcher.
	multiplier map[util.ContractType]int              // Stocks trade in units of 1, Options in units of 100.
	commission map[util.ContractType]map[string]int   // commission fees per type for base, unit.
	dispatcher *dispatcher.Dispatcher                 // My megaphone.
	adapter    interfaces.Adapter                     // Adapter already connected to "Broker".
	adptrAct   chan bool                              // Used to signal activity between Trader/Adapter.
}

func New(adapter interfaces.Adapter) *Trader {
	t := &Trader{}

	t.adapter = adapter
	t.adptrAct = make(chan bool, 1000)
	t.positions, _ = t.adapter.GetPositions()
	t.orders, _ = t.adapter.GetOrders("")

	t.pomIn = make(chan structs.ProtoOrderMessage, 1000)
	t.out = make(map[string]map[string]chan interface{})

	t.multiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	t.commission = map[util.ContractType]map[string]int{}
	t.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	t.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	t.dispatcher = dispatcher.New(1000)

	// Grind ProtoOrders into Orders.
	go func() {
		for pom := range t.pomIn {
			// Combine allotment with Path and send to Trader as ProtoOrder.
			o, err := t.constructOrder(pom.ProtoOrder)
			if err != nil {
				if pom.Reply != nil {
					pom.Reply <- pom.ProtoOrder
				}
				continue
			}

			// Submit order for execution.
			oid, err := t.submitOrder(o)
			if pom.Reply != nil {
				if err != nil {
					pom.Reply <- false
				} else {
					pom.Reply <- oid
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

func (t *Trader) constructOrder(po structs.ProtoOrder) (structs.Order, error) {
	o := structs.Order{Symbol: po.Path.Destination.Symbol, Type: po.Path.Destination.Type}
	o.Volume = (po.Allotment.Amount - t.commission[o.Type]["base"]) / (po.Path.LimitOpen * t.multiplier[o.Type])
	o.Limitprice = po.Path.LimitOpen
	o.Maxcost = (o.Volume * o.Limitprice * t.multiplier[o.Type]) + (o.Volume * t.commission[o.Type]["unit"])
	// Lazy search for acceptable volume.
	for o.Maxcost > (po.Allotment.Amount - t.commission[o.Type]["base"]) {
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
