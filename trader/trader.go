package trader

import (
	"github.com/eliwjones/thebox/destiny"
	"github.com/eliwjones/thebox/money"
	"github.com/eliwjones/thebox/util"

	"errors"
)

type ProtoOrder struct {
	Allotment money.Allotment // Money alloted for order.
	Path      destiny.Path    // How it wishes to "go out" and "return".
}

type Order struct {
	id         string            // Filled in if linked to Position.
	Symbol     string            // Whatever have to submit to api.
	volume     int               // How many of "it" do we want.
	limitprice int               // Price in cents to pay?  (And convert with api adapter?)
	_type      util.ContractType // STOCK, OPTION
	maxcost    int               // Expected maximum expenditure for order.
}

type Position struct {
	id        string // Some sort of id provided by api adapter?  (Thus can submit stop limit order for buytoclose).
	order     Order
	fillprice int // price per unit paid in cents.
}

type Delta struct {
}

type Trader struct {
	positions  []Position                             // Current outstanding positions.
	in         chan util.Message                      // Generally, ProtoOrders coming in.
	out        map[string]map[string]chan interface{} // Generally, Delta's heading out to dispatcher.
	multiplier map[util.ContractType]int
	commission map[util.ContractType]map[string]int // commission fees per type for base, unit.
}

func New(inBuf int64) *Trader {
	t := &Trader{}
	t.positions = []Position{}
	t.in = make(chan util.Message, inBuf)
	t.out = make(map[string]map[string]chan interface{})

	t.multiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	t.commission = map[util.ContractType]map[string]int{}
	t.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	t.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	// Proccess incoming trades, deltas.
	go func() {
		for message := range t.in {
			switch message.Data.(type) {
			case util.Subscription:
				s, _ := message.Data.(util.Subscription)
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
			case Delta:
				d, _ := message.Data.(Delta)
				for _, subscriber := range t.out["delta"] {
					subscriber <- d
				}
			}
		}
	}()

	return t
}

func (t *Trader) constructOrder(po ProtoOrder) (Order, error) {
	o := Order{Symbol: po.Path.Destination.Symbol, _type: po.Path.Destination.Type}
	o.volume = (po.Allotment.Amount - t.commission[o._type]["base"]) / (po.Path.LimitOpen * t.multiplier[o._type])
	o.limitprice = po.Path.LimitOpen
	o.maxcost = (o.volume * o.limitprice * t.multiplier[o._type]) + (o.volume * t.commission[o._type]["unit"])
	// Lazy search for acceptable volume.
	for o.maxcost > (po.Allotment.Amount - t.commission[o._type]["base"]) {
		o.volume--
		o.maxcost = (o.volume * o.limitprice * t.multiplier[o._type]) + (o.volume * t.commission[o._type]["unit"])
	}
	if o.volume == 0 {
		return o, errors.New("Impossible order. Not enough Allotment to cover commission.")
	}
	return o, nil
}

func (t *Trader) Subscribe(id string, whoami string, subscriber chan interface{}) {
	// Mainly to subscribe to deltas.
	s := util.Subscription{Id: id, Whoami: whoami, Subscriber: subscriber}
	t.in <- util.Message{Data: s}
}

func processOrder(o Order) (string, error) {
	// Ultimately, some thing that implements an interface will be used..
	return "dummyorderID", nil
}
