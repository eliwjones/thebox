package util

import (
	"time"
)

type Signal struct {
	Payload interface{}
	Wait    chan bool
}

type Delta struct {
}

type Trade struct {
	Allotment interface{}
	Path      interface{}
}

/*
type Position struct {
	Destination destinations.Destination // Where we have arrived.
	Price       int         // Price paid per unit volume (including commission) in cents.
	Volume      int         // Units purchased.
	ID          string      // Identifier for getting status or closing out.
}

type Trade struct {
	Allotment   Allotment   // Amount to allot.
	Destination Destination // Where should it go?
}
*/

var MS = func(time time.Time) int64 {
	return time.UnixNano() / 1000000
}

var Now = func() time.Time { return time.Now() }
