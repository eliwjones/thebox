package dispatcher

import (
	"github.com/eliwjones/thebox/destiny"
	"github.com/eliwjones/thebox/money"
	"github.com/eliwjones/thebox/trader"
	"github.com/eliwjones/thebox/util/structs"

	"testing"
)

func Test_Dispatcher_New(t *testing.T) {
	dstny := destiny.New(1 * 24 * 60 * 60 * 1000)
	d := New(1024, dstny)
	if d == nil {
		t.Errorf("Dispatcher New() call failed!")
	}
}

func Test_Dispatcher_Subscribe(t *testing.T) {
	dstny := destiny.New(1 * 24 * 60 * 60 * 1000)
	d := New(1024, dstny)

	tc := make(chan interface{}, 10)
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

func Test_Dispatcher_Allotment(t *testing.T) {
	dstny := destiny.New(1 * 24 * 60 * 60 * 1000)
	d := New(1024, dstny)

	// Subscribe to 'trade' channel.
	tc := make(chan interface{}, 10)
	d.Subscribe("trade", "tester", tc, true)

	// Add Path for Allotment.
	p := structs.Path{LimitClose: 1, LimitOpen: 2, Timestamp: 3}
	dstny.Put(p, true)

	a := money.Allotment{Amount: 100}
	reply := make(chan interface{})
	d.in <- structs.Message{Data: a, Reply: reply}

	response := <-reply
	if !response.(bool) {
		t.Errorf("Should have received 'true'")
	}
	o := <-tc
	if (o != trader.ProtoOrder{Allotment: a, Path: p}) {
		t.Errorf("Expected ProtoOrder: %+v, Got: %+v\n", trader.ProtoOrder{Allotment: a, Path: p}, o)
	}

	// Wipe out destiny.Paths and verify Allotment gets returned.
	d.destiny = destiny.New(1 * 24 * 60 * 60 * 1000)
	d.in <- structs.Message{Data: a, Reply: reply}
	response = <-reply
	if response != a {
		t.Errorf("No Path. Expected Allotment: %+v, Got: %+v", a, response)
	}
}

func Test_Dispatcher_Delta(t *testing.T) {
	dstny := destiny.New(1 * 24 * 60 * 60 * 1000)
	d := New(1024, dstny)

	// Subscribe to 'delta' channel.
	dc := make(chan interface{}, 10)
	d.Subscribe("delta", "tester", dc, true)

	delta := structs.Delta{}
	d.in <- structs.Message{Data: delta}
	ddelta := <-dc
	if ddelta != delta {
		t.Errorf("Expected delta: %+v, Got: %+v", delta, ddelta)
	}
}
