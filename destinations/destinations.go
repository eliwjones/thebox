package destinations

import (
	"errors"
	"github.com/eliwjones/thebox/structs"
	"github.com/eliwjones/thebox/util"
	"math/rand"
)

type Destination struct {
	Symbol string // "GOOG",  "GOOG1417Q525"
	Type   string // "Stock", "Option"
}

type TimestampedDestination struct {
	Destination Destination
	Timestamp   int64 // Millisecond Timestamp.
}

type Destinations struct {
	destinations []TimestampedDestination // Timestamped Destination so can decay.
	maxage       int64                    // Oldest Destination allowed.  // Should be a function?  For "easy" tuning?
	put          chan structs.Signal      // New Destinations come down this channel.
	decay        chan chan bool           // Process of decay has begun.
	decaying     bool                     // Currently decaying, so block put,get.
}

func (d *Destinations) Get() (Destination, error) {
	// Get random destination.  No locking since don't care? Maybe just a lock on len()?
	// May want to block while decaying?  Or, should just structure Decay to only happen after any possible Get()-ing?
	if len(d.destinations) == 0 {
		return Destination{}, errors.New("No Destinations")
	}
	td := d.destinations[rand.Intn(len(d.destinations))]
	return td.Destination, nil
}

func (d *Destinations) Put(destination Destination, block bool) {
	// Put new destination.
	var wait chan bool
	if block {
		wait = make(chan bool)
	}
	d.put <- structs.Signal{Payload: destination, Wait: wait}
	if block {
		<-wait
	}
}

func (d *Destinations) Decay() {
	// Initiate Decay process.
	wait := make(chan bool)
	d.decay <- wait
	<-wait
}

func NewDestinations(maxage int64) *Destinations {
	d := &Destinations{}
	d.destinations = []TimestampedDestination{}
	d.maxage = maxage
	d.put = make(chan structs.Signal, 100)
	d.decay = make(chan chan bool)
	d.decaying = false

	// Process Put, Decay calls.
	go func() {
		for {
			select {
			case signal := <-d.put:
				// Push TimestampedDestination.
				destination := signal.Payload.(Destination)
				if destination != (Destination{}) {
					d.destinations = append(d.destinations, TimestampedDestination{Destination: destination, Timestamp: util.MS(util.Now())})
				}
				if signal.Wait != nil {
					signal.Wait <- true
				}
			case wait := <-d.decay:
				d.decaying = true
				// Do some decaying.
				d.destinations = decay(d.destinations, d.maxage)
				d.decaying = false
				wait <- true
			}
		}
	}()

	return d
}

func decay(destinations []TimestampedDestination, maxage int64) []TimestampedDestination {
	now := util.MS(util.Now())
	newdestinations := []TimestampedDestination{}
	for _, td := range destinations {
		if td.Timestamp < now-maxage {
			continue
		}
		newdestinations = append(newdestinations, td)
	}
	return newdestinations
}
