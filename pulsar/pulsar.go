package pulsar

import (
	"github.com/eliwjones/thebox/dispatcher"

	"time"
)

type Pulsar struct {
	dispatcher *dispatcher.Dispatcher // Where to send 'pulses'
}

func New(period time.Duration) *Pulsar {
	p := &Pulsar{}
	p.dispatcher = dispatcher.New(10)
	ticker := time.NewTicker(period)

	go func() {
		for _ = range ticker.C {
			p.dispatcher.Send(true, "pulser")
		}

	}()

	return p
}

func (p *Pulsar) Subscribe(whoami string, subscriber chan interface{}) {
	p.dispatcher.Subscribe("pulser", whoami, subscriber, true)
}
