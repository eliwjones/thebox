package dispatcher

import (
	"github.com/eliwjones/thebox/util/structs"

	"testing"
)

func Test_Dispatcher_New(t *testing.T) {
	d := New(1024)
	if d == nil {
		t.Errorf("Dispatcher New() call failed!")
	}
}

func Test_Dispatcher_Subscribe(t *testing.T) {
	d := New(1024)

	tc := make(chan any, 10)
	d.Subscribe("trade", "tester", tc, true)
	_, exists := d.out["trade"]
	if !exists {
		t.Errorf("Dispatcher did not create 'trade' channel!")
	}
	_, exists = d.out["trade"]["tester"]
	if !exists {
		t.Errorf("Dispatcher did not create 'tester' subscription!")
	}

	// Another Subscription.
	d.Subscribe("trade", "tester2", tc, true)
	_, exists = d.out["trade"]["tester2"]
	if !exists {
		t.Errorf("Dispatcher did not create 'tester2' subscription!")
	}
}

func Test_Dispatcher_Send(t *testing.T) {
	d := New(1024)

	// Subscribe to 'delta' channel.
	ac := make(chan any, 10)
	d.Subscribe("allotment", "tester", ac, true)

	allotment := structs.Allotment{Amount: 1000}
	d.Send(allotment, "allotment")
	dallotment := <-ac
	if dallotment != allotment {
		t.Errorf("Expected delta: %+v, Got: %+v", allotment, dallotment)
	}
}
