package trader

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
)

type ProtoOrder struct {
	Allotment structs.Allotment // Money alloted for order.
	Path      structs.Path      // How it wishes to "go out" and "return".
}

type Trader struct {
	positions  []structs.Position                     // Current outstanding positions.
	in         chan structs.Message                   // Generally, ProtoOrders coming in.
	out        map[string]map[string]chan interface{} // Generally, Delta's heading out to dispatcher.
	multiplier map[util.ContractType]int
	commission map[util.ContractType]map[string]int // commission fees per type for base, unit.
}

func New(inBuf int) *Trader {
	t := &Trader{}
	t.positions = []structs.Position{}
	t.in = make(chan structs.Message, inBuf)
	t.out = make(map[string]map[string]chan interface{})

	t.multiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	t.commission = map[util.ContractType]map[string]int{}
	t.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	t.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	// Proccess incoming trades, deltas.
	go func() {
		for message := range t.in {
			switch message.Data.(type) {
			case structs.Subscription:
				s, _ := message.Data.(structs.Subscription)
				if t.out[s.Id] == nil {
					t.out[s.Id] = make(map[string]chan interface{})
				}
				t.out[s.Id][s.Whoami] = s.Subscriber
			case ProtoOrder:
				po, _ := message.Data.(ProtoOrder)
				o, err := t.constructOrder(po)
				if err != nil {
					// Can't construct order, send back.
					if message.Reply != nil {
						message.Reply <- po
					}
					continue
				}
				// Submit order for execution.
				processOrder(o)

				if message.Reply != nil {
					message.Reply <- true
				}
			case structs.Delta:
				d, _ := message.Data.(structs.Delta)
				for _, subscriber := range t.out["delta"] {
					subscriber <- d
				}
			}
		}
	}()

	return t
}

func (t *Trader) constructOrder(po ProtoOrder) (structs.Order, error) {
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

func (t *Trader) Subscribe(id string, whoami string, subscriber chan interface{}) {
	// Mainly to subscribe to deltas.
	s := structs.Subscription{Id: id, Whoami: whoami, Subscriber: subscriber}
	t.in <- structs.Message{Data: s}
}

func processOrder(o structs.Order) (string, error) {
	// Ultimately, some thing that implements an interface will be used..
	return "dummyorderID", nil
}
