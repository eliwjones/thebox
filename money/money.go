package money

import (
	"errors"
	"github.com/eliwjones/thebox/structs"
)

type Allotment struct {
	Amount int // Some parcel of total value in cents.
}

type Delta struct {
	Amount  int     // Return in cents.
	Percent float32 // Delta.Amount/(Position.Price*Position.Volume)
}

type Money struct {
	Total      int                 // Total money in cents.
	Available  int                 // Available money in cents.
	Allotments []Allotment         // Currently available Allotments.
	Deltas     []Delta             // Current bits of Delta.
	get        chan chan Allotment // Request allotment.
	put        chan structs.Signal // Put allotment.
	reallot    chan chan bool      // Re-balance Allotments.
}

func (m *Money) Get() (Allotment, error) {
	var err error
	reply := make(chan Allotment)
	m.get <- reply
	allotment := <-reply
	if allotment == (Allotment{}) {
		err = errors.New("No Allotments")
	}
	return allotment, err
}

func (m *Money) Put(allotment Allotment, block bool) {
	var wait chan bool
	if block {
		wait = make(chan bool)
	}
	m.put <- structs.Signal{Payload: allotment, Wait: wait}
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
	m.put = make(chan structs.Signal, 100)
	m.reallot = make(chan chan bool, 10)

	// Process Get,Put, ReAllot calls.
	go func() {
		for {
			select {
			case c := <-m.get:
				// Pop Allotment from m.Allotments and send it down c.
				allotment := Allotment{}
				if len(m.Allotments) > 0 {
					allotment, m.Allotments = m.Allotments[len(m.Allotments)-1], m.Allotments[:len(m.Allotments)-1]
					m.Available -= allotment.Amount
				}
				c <- allotment
			case signal := <-m.put:
				// Push Allotment to m.Allotments.
				allotment := signal.Payload.(Allotment)
				// Don't want empty Allotments.
				if allotment != (Allotment{}) {
					m.Allotments = append(m.Allotments, allotment)
					m.Available += allotment.Amount
				}
				if signal.Wait != nil {
					signal.Wait <- true
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
