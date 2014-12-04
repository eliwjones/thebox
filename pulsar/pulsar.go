package pulsar

import (
	"github.com/eliwjones/thebox/dispatcher"

	"os"
	"sort"
	"strconv"
)

type Pulsar struct {
	dispatcher *dispatcher.Dispatcher // Where to send 'pulses'
	pulses     []int64                // tape of pulses to send out.
	replies    map[string]chan int64  // To synchronize, await replies.
}

func New(datadir string, start string, stop string) *Pulsar {
	p := &Pulsar{}
	p.dispatcher = dispatcher.New(10)
	p.pulses = loadPulses(datadir, start, stop)
	p.replies = map[string]chan int64{}

	return p
}

func (p *Pulsar) Start() {
	for _, pulse := range p.pulses {
		p.dispatcher.Send(pulse, "pulser")
		// Feels wrong, but don't feel like rolling await-reply functionality into Dispatcher.
		for _, reply := range p.replies {
			<-reply
		}

	}
	// Send -1 as shutdown signal.
	p.dispatcher.Send(int64(-1), "pulser")
	for _, reply := range p.replies {
		<-reply
	}
}

func (p *Pulsar) Subscribe(whoami string, subscriber chan interface{}, reply chan int64) {
	p.dispatcher.Subscribe("pulser", whoami, subscriber, true)
	p.replies[whoami] = reply
}

func loadPulses(datadir string, start string, stop string) []int64 {
	// Inefficient, but to the point.
	// Panic-y since have no desire to run if pulses are suspect.
	d, err := os.Open(datadir)
	if err != nil {
		panic(err)
	}
	ps, err := d.Readdirnames(-1)
	if err != nil {
		panic(err)
	}
	sort.Strings(ps)
	pulses := []int64{}
	for _, p := range ps {
		if p > stop {
			continue
		}
		if p < start {
			continue
		}
		pulse, err := strconv.ParseFloat(p, 64)
		if err != nil {
			panic(err)
		}
		pulses = append(pulses, int64(pulse))
	}
	return pulses
}
