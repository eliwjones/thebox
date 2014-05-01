package thebox

import (
	"errors"
	"math/rand"
	"time"
)

type Signal struct {
	payload interface{}
	wait    chan bool
}

type Allotment struct {
	Amount int // Some parcel of total value in cents.
}

type Delta struct {
	Amount  int     // Return in cents.
	Percent float32 // Delta.Amount/(Position.Price*Position.Volume)
}

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
	put          chan Signal              // New Destinations come down this channel.
	decay        chan chan bool           // Process of decay has begun.
	decaying     bool                     // Currently decaying, so block put,get.
}

type Trade struct {
	Allotment   Allotment   // Amount to allot.
	Destination Destination // Where should it go?
}

type Position struct {
	Destination Destination // Where we have arrived.
	Price       int         // Price paid per unit volume (including commission) in cents.
	Volume      int         // Units purchased.
	ID          string      // Identifier for getting status or closing out.
}

type Money struct {
	Total      int                 // Total money in cents.
	Available  int                 // Available money in cents.
	Allotments []Allotment         // Currently available Allotments.
	Deltas     []Delta             // Current bits of Delta.
	get        chan chan Allotment // Request allotment.
	put        chan Signal         // Put allotment.
	reallot    chan chan bool      // Re-balance Allotments.
}

func (m *Money) Get() Allotment {
	allotment := make(chan Allotment)
	m.get <- allotment
	return <-allotment
}

func (m *Money) Put(allotment Allotment, block bool) {
	var wait chan bool
	if block {
		wait = make(chan bool)
	}
	m.put <- Signal{payload: allotment, wait: wait}
	if block {
		<-wait
	}
}

func (m *Money) ReAllot() {
	wait := make(chan bool)
	m.reallot <- wait
	<-wait
}

// http://golang.org/doc/codewalk/sharemem/
// For "idiom" on controlling access to shared map/slice.
// Other: https://gist.github.com/deckarep/7685352

func NewMoney(cash int) *Money {
	m := &Money{}
	m.Total = cash
	m.Available = cash
	m.Allotments = []Allotment{}
	m.Deltas = []Delta{}
	m.get = make(chan chan Allotment, 100)
	m.put = make(chan Signal, 100)
	m.reallot = make(chan chan bool, 10)

	// Process Get,Put, ReAllot calls.
	go func() {
		for {
			var allotment Allotment
			select {
			case c := <-m.get:
				// Pop Allotment from m.Allotments and send it down c.
				allotment, m.Allotments = m.Allotments[len(m.Allotments)-1], m.Allotments[:len(m.Allotments)-1]
				m.Available -= allotment.Amount
				c <- allotment
			case signal := <-m.put:
				// Push Allotment to m.Allotments.
				allotment = signal.payload.(Allotment)
				m.Allotments = append(m.Allotments, allotment)
				m.Available += allotment.Amount
				if signal.wait != nil {
					signal.wait <- true
				}
			case wait := <-m.reallot:
				m.Allotments = reallot(m.Available)
				wait <- true
			}
		}
	}()

	// Create Initial Allotments.
	m.ReAllot()

	return m
}

// Mindless allocation of 1% Allotments.
func reallot(cash int) []Allotment {
	allotments := []Allotment{}
	allotment := Allotment{}
	allotment.Amount = cash / 100
	for i := 0; i < 100; i++ {
		allotments = append(allotments, allotment)
	}
	return allotments
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
	d.put <- Signal{payload: destination, wait: wait}
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
	d.put = make(chan Signal, 100)
	d.decay = make(chan chan bool)
	d.decaying = false

	// Process Put, Decay calls.
	go func() {
		for {
			select {
			case signal := <-d.put:
				// Push TimestampedDestination.
				destination := signal.payload.(Destination)
				d.destinations = append(d.destinations, TimestampedDestination{Destination: destination, Timestamp: MS(Now())})
				if signal.wait != nil {
					signal.wait <- true
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
	now := MS(Now())
	newdestinations := []TimestampedDestination{}
	for _, td := range destinations {
		if td.Timestamp < now-maxage {
			continue
		}
		newdestinations = append(newdestinations, td)
	}
	return newdestinations
}

// Utilities.
var MS = func(time time.Time) int64 {
	return time.UnixNano() / 1000000
}

var Now = func() time.Time { return time.Now() }
