package dispatcher

import (
	"github.com/eliwjones/thebox/destiny"
	"github.com/eliwjones/thebox/money"
	"github.com/eliwjones/thebox/trader"
	"github.com/eliwjones/thebox/util"
)

type Dispatcher struct {
	in      chan util.Message                      // Something comes in.
	out     map[string]map[string]chan interface{} // Send things out to whoever wants "it".
	destiny *destiny.Destiny                       // Place to get my paths from.
}

func New(inBuf int64, dstny *destiny.Destiny) *Dispatcher {
	d := &Dispatcher{}
	d.in = make(chan util.Message, inBuf)
	d.out = make(map[string]map[string]chan interface{})
	d.destiny = dstny

	// Run go func() to process the old in and out.
	go func() {
		for message := range d.in {
			switch message.Data.(type) {
			case util.Subscription:
				// Subscriptions are fairly sparse, so no need for separate channel.
				s, _ := message.Data.(util.Subscription)
				if d.out[s.Id] == nil {
					d.out[s.Id] = make(map[string]chan interface{})
				}
				d.out[s.Id][s.Whoami] = s.Subscriber
			case money.Allotment:
				allotment, _ := message.Data.(money.Allotment)
				path, err := d.destiny.Get()
				if err != nil {
					// Could not get path so return allotment.
					if message.Reply != nil {
						message.Reply <- allotment
					}
					continue
				}
				for _, subscriber := range d.out["trade"] {
					subscriber <- trader.ProtoOrder{Allotment: allotment, Path: path}
				}
				if message.Reply != nil {
					message.Reply <- true
				}
			case trader.Delta:
				// Handle Delta.
				delta, _ := message.Data.(trader.Delta)
				for _, subscriber := range d.out["delta"] {
					subscriber <- delta
				}
			}
		}
	}()

	return d
}

func (d *Dispatcher) Subscribe(id string, whoami string, subscriber chan interface{}) {
	// Send subscription to d.in for processing.
	s := util.Subscription{Id: id, Whoami: whoami, Subscriber: subscriber}
	d.in <- util.Message{Data: s}
}
