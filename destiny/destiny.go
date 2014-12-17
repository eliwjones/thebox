package destiny

import (
	"github.com/eliwjones/thebox/dispatcher"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
	"math"
	"math/rand"
)

type Destiny struct {
	destinations []structs.Destination         // Need those destinations.
	paths        []structs.Path                // Timestamped paths to destinations..
	maxage       int64                         // Oldest Path allowed.  // Should be a function?  For "easy" tuning?
	put          chan structs.Signal           // New Paths come down this channel.
	decay        chan chan bool                // Process of decay has begun.
	decaying     bool                          // Currently decaying, so block put,get.
	dispatcher   *dispatcher.Dispatcher        // My megaphone.
	amIn         chan structs.AllotmentMessage // Allotments come in here.
	dIn          chan structs.Delta            // Deltas come in here.
}

func (d *Destiny) Get() (structs.Path, error) {
	// Get random Path.  No locking since don't care? Maybe just a lock on len()?
	// May want to block while decaying?  Or, should just structure Decay to only happen after any possible Get()-ing?
	if len(d.paths) == 0 {
		return structs.Path{}, errors.New("No Paths to Destinations.")
	}
	path := d.paths[rand.Intn(len(d.paths))]
	return path, nil
}

func (d *Destiny) Put(path structs.Path, block bool) {
	// Put new path.
	var wait chan bool
	if block {
		wait = make(chan bool)
	}
	d.put <- structs.Signal{Payload: path, Wait: wait}
	if block {
		<-wait
	}
}

func (d *Destiny) Decay() {
	// Initiate Decay process.
	wait := make(chan bool)
	d.decay <- wait
	<-wait
}

func New(maxage int64) *Destiny {
	d := &Destiny{}
	d.paths = []structs.Path{}
	d.maxage = maxage
	d.put = make(chan structs.Signal, 100)
	d.decay = make(chan chan bool)
	d.decaying = false

	d.dispatcher = dispatcher.New(1000)
	d.amIn = make(chan structs.AllotmentMessage, 1000)
	d.dIn = make(chan structs.Delta, 1000)

	// Process Put, Decay calls.
	go func() {
		for {
			select {
			case signal := <-d.put:
				// Push Path.
				path := signal.Payload.(structs.Path)
				if path != (structs.Path{}) {
					d.paths = append(d.paths, path)
				}
				if signal.Wait != nil {
					signal.Wait <- true
				}
			case wait := <-d.decay:
				d.decaying = true
				// Do some decaying.
				d.paths = decay(d.paths, d.maxage)
				d.decaying = false
				wait <- true
			}
		}
	}()

	// Grind Allotments into ProtoOrders.
	go func() {
		for am := range d.amIn {
			// Combine allotment with Path and send to Trader as ProtoOrder.
			p, err := d.Get()
			if err != nil {
				if am.Reply != nil {
					am.Reply <- am.Allotment
				}
				continue
			}

			po := structs.ProtoOrder{Allotment: am.Allotment, Path: p}
			d.dispatcher.Send(po, "protoorder")

			if am.Reply != nil {
				am.Reply <- true
			}
		}
	}()

	// Process Deltas.
	go func() {
		for delta := range d.dIn {
			// Dummy hueristic, use better.
			pathModifier(delta, d)
		}
	}()

	return d
}

func decay(paths []structs.Path, maxage int64) []structs.Path {
	now := funcs.MS(funcs.Now())
	newpaths := []structs.Path{}
	for _, path := range paths {
		if path.Timestamp < now-maxage {
			continue
		}
		newpaths = append(newpaths, path)
	}
	return newpaths
}

func pathModifier(delta structs.Delta, d *Destiny) {
	if delta.Percent > 100 {
		d.Put(delta.Path, false)
	}
}

// Protean Penalty Functions.

func distanceFromStrikePenalty(strike int, maxStrike, underlyingPrice int, k float64) float64 {
	// Not sure of least messy way to write out function.

	strikeDistance := math.Abs(float64(strike - underlyingPrice))
	maxDistance := math.Abs(float64(maxStrike - underlyingPrice))

	penalty := k * (strikeDistance / maxDistance)

	return penalty
}

func distanceFromExpirationPenalty(utcTimestamp int64, expirationTimestamp int64, k float64) float64 {
	secondsInDay := int64(24 * 60 * 60)
	maxDistance := float64(4 * secondsInDay)

	currentDistance := float64(utcTimestamp - (expirationTimestamp - secondsInDay))
	if currentDistance > 0 {
		// We are on expiration day, no penalty for nothing.
		return 0
	}
	currentDistance = math.Abs(currentDistance)

	penalty := k * (currentDistance / maxDistance)

	return penalty
}

func premiumPenalty(optionPrice int, closestPrice int, farthestPrice int, k float64) float64 {
	currentDistance := math.Abs(float64(optionPrice - farthestPrice))
	maxDistance := math.Abs(float64(closestPrice - farthestPrice))

	penalty := k * (currentDistance / maxDistance)

	return penalty
}

// Probability of adding option to list of paths?
//     (1-distanceFromStrikePenalty() + 1-distanceFromExpirationPenalty())/2 - premiumPenalty()
//
// This of course, is a big deviation from initial thinking...
// Not sure of least dumb way to convert this to list of N paths.
