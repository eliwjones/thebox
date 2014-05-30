package pulsar

import (
	"github.com/eliwjones/thebox/dispatcher"

	"time"
)

var periods map[int]time.Duration = map[int]time.Duration{
	1:     1 * time.Second,
	1000:  1 * time.Millisecond,
	5000:  200 * time.Microsecond, // Occassionally drops tics.
	10000: 100 * time.Microsecond} // Seems to be max before dropping "too many" tics.

type Pulsar struct {
	now        int64                  // "Milliseconds" since Epoch.
	period     time.Duration          // What is my current duration?
	dispatcher *dispatcher.Dispatcher // Where to send 'pulses'
}

func New(startms int64, speedup int) *Pulsar {
	p := &Pulsar{}
	p.now = startms
	p.period = periods[speedup]
	p.dispatcher = dispatcher.New(10)
	ticker := time.NewTicker(p.period)

	go func() {
		for _ = range ticker.C {
			p.dispatcher.Send(p.now, "pulser")
			p.now += 1000
		}

	}()

	return p
}

func (p *Pulsar) Subscribe(whoami string, subscriber chan interface{}) {
	p.dispatcher.Subscribe("pulser", whoami, subscriber, true)
}
