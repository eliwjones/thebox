package dispatcher

import (
	"github.com/eliwjones/thebox/destiny"
	"github.com/eliwjones/thebox/money"

	"fmt"
	"testing"
)

func Test_Dispatcher(t *testing.T) {
	dstny := destiny.New(int64(1 * 24 * 60 * 60 * 1000))
	path := destiny.Path{LimitClose: "1", LimitOpen: "2", Timestamp: 3}
	dstny.Put(path, true)

	dispatcher := New(1024, dstny)

	tradeChannel := make(chan interface{}, 10)
	dispatcher.Subscribe("trade", "tester", tradeChannel)

	allotment := money.Allotment{Amount: 100}
	dispatcher.in <- Message{Data: allotment}

	trade := <-tradeChannel
	// Should have received Trade{Allotment, Path}
	fmt.Printf("%+v\n", trade)
}
