package structs

import (
	"github.com/eliwjones/thebox/util"
)

type Allotment struct {
	Amount int // Some parcel of total value in cents.
}

type AllotmentMessage struct {
	Allotment Allotment
	Reply     chan interface{}
}

type Maximum struct {
	// Fields used for Key-ing mapmapmap (or writing to file).
	Expiration   string
	OptionSymbol string
	Timestamp    int64
	Underlying   string

	MaximumBid    int
	OptionAsk     int
	OptionBid     int
	OptionType    string
	Strike        int
	UnderlyingBid int
	Volume        int
	MaxTimestamp  int64
}

// For now, intuitively setting all prices to cents.
// Better not forget to convert to dollars on submission!
type Option struct {
	Expiration string
	Strike     int
	Symbol     string
	Time       int64
	Type       string

	Ask          int
	Bid          int
	IV           float64
	Last         int
	OpenInterest int
	Underlying   string
	Volume       int
}

type Order struct {
	Id         string            // Filled in if linked to Position.
	Symbol     string            // Whatever have to submit to api.
	Volume     int               // How many of "it" do we want.
	Limitprice int               // Price in cents to pay?  (And convert with api adapter?)
	Type       util.ContractType // STOCK, OPTION
	Maxcost    int               // Expected maximum expenditure for order.
	ProtoOrder ProtoOrder        // Needed for ultimate Delta calculation? (Maybe just loosely associate by id)
}

type Position struct {
	Id        string // Some sort of id provided by api adapter?  (Thus can submit stop limit order for buytoclose).
	Order     Order  // Order that position originated from.
	Fillprice int    // price per unit paid in cents.
}

type ProtoOrder struct {
	LimitOpen  int               // Set by Destiny from chosen edge.
	Symbol     string            // "GOOG", "GOOG_030615C620"
	Timestamp  int64             // Suppose they may could expire..?
	Type       util.ContractType // util.OPTION, util.STOCK
	Underlying string            // Tacking this in here to facilitate Trader GetQuote() lookups.

	Reply chan interface{} `json:"-"`
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

type Stock struct {
	Ask    int
	Bid    int
	High   int
	Last   int
	Low    int
	Symbol string
	Time   int64 // Seconds in HH:MM:SS.
	Volume int
}

type Message struct {
	Data  interface{}      // Shall this be an interface?
	Reply chan interface{} // Reply if needed..
}
