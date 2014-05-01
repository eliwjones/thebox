package thebox

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
	put        chan MoneySignal    // Put allotment.
	reallot    chan chan bool      // Re-balance Allotments.
}

type MoneySignal struct {
	payload Allotment
	wait    chan bool
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
	m.put <- MoneySignal{payload: allotment, wait: wait}
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
	m.put = make(chan MoneySignal, 100)
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
			case moneysignal := <-m.put:
				// Push Allotment to m.Allotments.
				m.Allotments = append(m.Allotments, moneysignal.payload)
				m.Available += moneysignal.payload.Amount
				if moneysignal.wait != nil {
					moneysignal.wait <- true
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
