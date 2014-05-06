package destiny

import (
	"errors"
	"github.com/eliwjones/thebox/util"
	"math/rand"
)

type Destination struct {
	Underlying string            // "GOOG",  "SPX",  // Underlying?
	Symbol     string            // function of Underlying? f(d.Underlying, 1, d.Type)
	Type       util.ContractType // util.OPTION, util.STOCK
}

type Path struct {
	Destination Destination
	LimitOpen   int // populated by function of current (Bid, Ask)?  Too specific??
	LimitClose  int // populated by function of LimitOpen?
	Timestamp   int64
}

type Destiny struct {
	destinations []Destination    // Need those destinations.
	paths        []Path           // Timestamped paths to destinations..
	maxage       int64            // Oldest Path allowed.  // Should be a function?  For "easy" tuning?
	put          chan util.Signal // New Paths come down this channel.
	decay        chan chan bool   // Process of decay has begun.
	decaying     bool             // Currently decaying, so block put,get.
}

func (d *Destiny) Get() (Path, error) {
	// Get random Path.  No locking since don't care? Maybe just a lock on len()?
	// May want to block while decaying?  Or, should just structure Decay to only happen after any possible Get()-ing?
	if len(d.paths) == 0 {
		return Path{}, errors.New("No Paths to Destinations.")
	}
	path := d.paths[rand.Intn(len(d.paths))]
	return path, nil
}

func (d *Destiny) Put(path Path, block bool) {
	// Put new path.
	var wait chan bool
	if block {
		wait = make(chan bool)
	}
	d.put <- util.Signal{Payload: path, Wait: wait}
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
	d.paths = []Path{}
	d.maxage = maxage
	d.put = make(chan util.Signal, 100)
	d.decay = make(chan chan bool)
	d.decaying = false

	// Process Put, Decay calls.
	go func() {
		for {
			select {
			case signal := <-d.put:
				// Push Path.
				path := signal.Payload.(Path)
				if path != (Path{}) {
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

	return d
}

func decay(paths []Path, maxage int64) []Path {
	now := util.MS(util.Now())
	newpaths := []Path{}
	for _, path := range paths {
		if path.Timestamp < now-maxage {
			continue
		}
		newpaths = append(newpaths, path)
	}
	return newpaths
}
