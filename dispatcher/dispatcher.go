package dispatcher

import (
	"github.com/eliwjones/thebox/util/structs"
)

type Dispatcher struct {
	in  chan structs.Message           // Something comes in.
	out map[string]map[string]chan any // Send things out to whoever wants "it".
}

func New(inBuf int) *Dispatcher {
	d := &Dispatcher{}
	d.in = make(chan structs.Message, inBuf)
	d.out = make(map[string]map[string]chan any)

	// Run go func() to process the old in and out.
	// switch/case maybe outdated since only accept Subscribes.
	go func() {
		for message := range d.in {
			switch message.Data.(type) {
			case structs.Subscription:
				// Not really sure if need "wait" for subscription.
				s, _ := message.Data.(structs.Subscription)
				if d.out[s.Id] == nil {
					d.out[s.Id] = make(map[string]chan any)
				}
				d.out[s.Id][s.Whoami] = s.Subscriber
				if message.Reply != nil {
					message.Reply <- true
				}
			}
		}
	}()

	return d
}

func (d *Dispatcher) Subscribe(id string, whoami string, subscriber chan any, wait bool) {
	// Send subscription to d.in for processing.
	var r chan any
	s := structs.Subscription{Id: id, Whoami: whoami, Subscriber: subscriber}
	m := structs.Message{Data: s}
	if wait {
		r = make(chan any)
		m.Reply = r
	}
	d.in <- m
	if wait {
		<-r
	}
}

func (d *Dispatcher) Send(message any, id string) {
	// Not sure want to worry about concurrent access here?
	for _, subscriber := range d.out[id] {
		subscriber <- message
	}
}
