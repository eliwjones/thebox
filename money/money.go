package money

import (
	"github.com/eliwjones/thebox/dispatcher"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
	"math/rand"
)

type Money struct {
	Total       int                         // Total money in cents.
	Available   int                         // Available money in cents.
	Allotments  []structs.Allotment         // Currently available Allotments.
	deltaSum    int                         // Sum of Delta amounts.
	allotmentIn chan structs.Allotment      // Deltas rolling in.
	get         chan chan structs.Allotment // Request allotment.
	put         chan structs.Signal         // Put allotment.
	reallot     chan chan bool              // Re-balance Allotments.
	dispatcher  *dispatcher.Dispatcher      // My megaphone.
}

func (m *Money) Get() (structs.Allotment, error) {
	var err error
	reply := make(chan structs.Allotment)
	m.get <- reply
	allotment := <-reply
	if allotment == (structs.Allotment{}) {
		err = errors.New("no allotments")
	}
	return allotment, err
}

func (m *Money) Put(allotment structs.Allotment, block bool) {
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

func (m *Money) getRandomAllotment() (a structs.Allotment, err error) {
	// Insane? Recovering from panic of non-existent index.
	// Golang Try/Catch?
	defer func() {
		r := recover()
		if r != nil {
			amt := m.Total / 100
			a = structs.Allotment{Amount: amt}
			if amt <= 0 {
				err = errors.New("not enough total value to construct allotment")
			}
		}
	}()
	a = m.Allotments[rand.Intn(len(m.Allotments))]
	return a, err
}

func New(cash int) *Money {
	m := &Money{}

	m.Total = cash
	m.Available = m.Total

	m.Allotments = []structs.Allotment{}
	m.deltaSum = 0
	m.allotmentIn = make(chan structs.Allotment, 100)
	m.get = make(chan chan structs.Allotment, 100)
	m.put = make(chan structs.Signal, 100)
	m.reallot = make(chan chan bool, 10)

	m.dispatcher = dispatcher.New(1000)

	// Send any mod 100 remainder to Deltas.

	// Process Get,Put, ReAllot calls.
	go func() {
		for {
			select {
			case c := <-m.get:
				// Pop Allotment from m.Allotments and send it down c.
				allotment := structs.Allotment{}
				if len(m.Allotments) > 0 {
					allotment, m.Allotments = m.Allotments[len(m.Allotments)-1], m.Allotments[:len(m.Allotments)-1]
					m.Available -= allotment.Amount
				}
				c <- allotment
			case signal := <-m.put:
				// Push Allotment to m.Allotments.
				allotment := signal.Payload.(structs.Allotment)
				// Don't want empty Allotments.
				if allotment != (structs.Allotment{}) {
					m.Allotments = append(m.Allotments, allotment)
					m.Available += allotment.Amount
					// Trivial to see that excess allotments come in means more Total.
					// How to infer reduction of Total?  Or is that always had by external call.
					if m.Available > m.Total {
						m.Total = m.Available
					}
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

	// Process incoming Deltas
	go func() {
		for allotment := range m.allotmentIn {
			m.deltaSum += allotment.Amount

			// Determine if can construct Allotment and send on.
			a, err := m.getRandomAllotment()
			for m.deltaSum >= a.Amount && err == nil {
				m.deltaSum -= a.Amount
				m.Put(a, false)
			}
		}
	}()

	// Create Initial Allotments.
	m.ReAllot()

	return m
}

// Mindless allocation of 1% Allotments.
func reallot(cash int) []structs.Allotment {
	allotments := []structs.Allotment{}
	allotment := structs.Allotment{}
	allotment.Amount = cash / 100
	if allotment.Amount <= 0 {
		return allotments
	}
	for range 100 {
		allotments = append(allotments, allotment)
	}
	return allotments
}
