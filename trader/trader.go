package trader

import (
	"github.com/eliwjones/thebox/dispatcher"
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
)

type Trader struct {
	positions  []structs.Position                     // Current outstanding positions.
	pomIn      chan structs.ProtoOrderMessage         // Generally, ProtoOrders coming in.
	out        map[string]map[string]chan interface{} // Generally, Delta's heading out to dispatcher.
	multiplier map[util.ContractType]int              // Stocks trade in units of 1, Options in units of 100.
	commission map[util.ContractType]map[string]int   // commission fees per type for base, unit.
	dispatcher *dispatcher.Dispatcher                 // My megaphone.
}

func New(inBuf int) *Trader {
	t := &Trader{}
	t.positions = []structs.Position{}
	t.pomIn = make(chan structs.ProtoOrderMessage, inBuf)
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
			oid, err := processOrder(o)
			if pom.Reply != nil {
				if err != nil {
					pom.Reply <- false
				} else {
					pom.Reply <- oid
				}
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

func processOrder(o structs.Order) (string, error) {
	// Ultimately, some thing that implements an interface will be used..
	return "dummyorderID", nil
}
