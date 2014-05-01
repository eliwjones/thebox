package destinations

import (
	"github.com/eliwjones/thebox/util"
	"testing"
	"time"
)

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

	now := util.MS(util.Now())
	// Mock MS() to give time beyond maxage.
	util.MS = func(time time.Time) int64 { return now + destinations.maxage + 10 }

	destination = Destination{Symbol: "AAPL", Type: "stock"}
	destinations.Put(destination, true)
	if len(destinations.destinations) != 2 {
		t.Errorf("Should be 2 destinations, but there are %d!", len(destinations.destinations))
	}

	destinations.Decay()
	if len(destinations.destinations) != 1 {
		t.Errorf("Should be 1 destination now, but there are %d!", len(destinations.destinations))
	}

	// Verify cannot add empty Destination.
	length := len(destinations.destinations)
	destinations.Put(Destination{}, true)
	if len(destinations.destinations) != length {
		t.Errorf("Empty Destination should not have been added!")
	}
}
