package thebox

import (
	"testing"
	"time"
)

func Test_Money(t *testing.T) {
	cash := 1000000 * 100 // Million $ in cents.
	money := NewMoney(cash)
	money.ReAllot()

	total := len(money.Allotments)
	allotment := money.Get()

	if len(money.Allotments) != total-1 {
		t.Errorf("Len of Allotments should be %d, but got: %d", total-1, len(money.Allotments))
	}

	if money.Available != money.Total-allotment.Amount {
		t.Errorf("Should be less Available than Total.  Available: %d", money.Available)
	}

	money.Put(allotment, true)

	if money.Available != money.Total {
		t.Errorf("Available money should equal Total money! Available: %d", money.Available)
	}
}

func Test_Destinations(t *testing.T) {
	onedayinmilliseconds := int64(1 * 24 * 60 * 60 * 1000)
	destinations := NewDestinations(onedayinmilliseconds)

	if len(destinations.destinations) != 0 {
		t.Errorf("Should be 0 destinations, but there are %d!", len(destinations.destinations))
	}

	destination, err := destinations.Get()
	if err == nil {
		t.Errorf("Should have received error, but got destination: %v, err: %s", destination, err)
	}

	destination = Destination{Symbol: "GOOG", Type: "stock"}
	destinations.Put(destination, true)
	if len(destinations.destinations) != 1 {
		t.Errorf("Should be 1 destination, but there are %d!", len(destinations.destinations))
	}

	destination, err = destinations.Get()
	if destination.Symbol != "GOOG" || destination.Type != "stock" {
		t.Errorf("What have you done? Expected 'GOOG','stock' but received: %s, %s!", destination.Symbol, destination.Type)
	}

	now := MS(Now())
	// Mock MS() to give time beyond maxage.
	MS = func(time time.Time) int64 { return now + destinations.maxage + 10 }

	destination = Destination{Symbol: "AAPL", Type: "stock"}
	destinations.Put(destination, true)
	if len(destinations.destinations) != 2 {
		t.Errorf("Should be 2 destinations, but there are %d!", len(destinations.destinations))
	}

	destinations.Decay()
	if len(destinations.destinations) != 1 {
		t.Errorf("Should be 1 destination now, but there are %d!", len(destinations.destinations))
	}
}
