package dispatcher

import (
	"github.com/eliwjones/thebox/destiny"
	"github.com/eliwjones/thebox/money"
	"github.com/eliwjones/thebox/util"

	"testing"
)

func Test_Dispatcher(t *testing.T) {
	dstny := destiny.New(int64(1 * 24 * 60 * 60 * 1000))
	dispatcher := New(1024, dstny)

	path := destiny.Path{LimitClose: "1", LimitOpen: "2", Timestamp: 3}
	dstny.Put(path, true)

	tradeChannel := make(chan interface{}, 10)
	dispatcher.Subscribe("trade", "tester", tradeChannel)

	allotment := money.Allotment{Amount: 100}
	reply := make(chan interface{})
	dispatcher.in <- Message{Data: allotment, Reply: reply}

	response := <-reply
	if !response.(bool) {
		t.Errorf("Should have received 'true'")
	}

	trade := <-tradeChannel
	if (trade != util.Trade{Allotment: allotment, Path: path}) {
		t.Errorf("Expected Trade with allotment, path but got Trade: %+v\n", trade)
	}

	// Test sending again.
	dispatcher.in <- Message{Data: allotment, Reply: reply}
	response = <-reply
	if !response.(bool) {
		t.Errorf("Should have received 'true'")
	}

	// Decay paths. Now should receive allotment back in reply.
	dstny.Decay()
	_, err := dstny.Get()
	if err == nil {
		t.Errorf("Expected error since there should be no Paths to Get().")
	}
	dispatcher.in <- Message{Data: allotment, Reply: reply}
	response = <-reply
	if response.(money.Allotment) != allotment {
		t.Errorf("Should have received allotment back since there was no Path.")
	}
}
