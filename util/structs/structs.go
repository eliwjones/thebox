package structs

import (
	"github.com/eliwjones/thebox/util"
)

type Delta struct {
	Amount  int
	Percent float32
	Path    Path
}

type Destination struct {
	Underlying string            // "GOOG",  "SPX",  // Underlying?
	Symbol     string            // function of Underlying? f(d.Underlying, 1, d.Type)
	Type       util.ContractType // util.OPTION, util.STOCK
}

type Order struct {
	Id         string            // Filled in if linked to Position.
	Symbol     string            // Whatever have to submit to api.
	Volume     int               // How many of "it" do we want.
	Limitprice int               // Price in cents to pay?  (And convert with api adapter?)
	Type       util.ContractType // STOCK, OPTION
	Maxcost    int               // Expected maximum expenditure for order.
}

type Path struct {
	Destination Destination
	LimitOpen   int // populated by function of current (Bid, Ask)?  Too specific??
	LimitClose  int // populated by function of LimitOpen?
	Timestamp   int64
}

type Position struct {
	Id        string // Some sort of id provided by api adapter?  (Thus can submit stop limit order for buytoclose).
	Order     Order  // Order that position originated from.
	Fillprice int    // price per unit paid in cents.
}

type Subscription struct {
	Id         string           // What is id of thing you are subscribing to.
	Whoami     string           // Who are you in case we need to delete.
	Subscriber chan interface{} // Where to send info.
}

type Signal struct {
	Payload interface{}
	Wait    chan bool
}

type Message struct {
	Data  interface{}      // Shall this be an interface?
	Reply chan interface{} // Reply if needed..
}
