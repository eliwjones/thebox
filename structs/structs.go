package structs

type Signal struct {
	Payload interface{}
	Wait    chan bool
}

// Not sure where these structs will go in future.. so for now, they sit here.
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
