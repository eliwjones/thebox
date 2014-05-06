package trader

import (
	"github.com/eliwjones/thebox/destiny"
	"github.com/eliwjones/thebox/money"
	"testing"
)

func Test_Trader(t *testing.T) {
	trdr := New(10)

	po := ProtoOrder{Allotment: money.Allotment{Amount: 1000 * 100}, Path: destiny.Path{}}
	po.Path.Destination.Symbol = "GOOG May 2014 1234 Put"
	po.Path.Destination.Type = OPTION
	po.Path.LimitOpen = 990           // $9.90 option.

	order, err := trdr.constructOrder(po)
	if err == nil {
		t.Errorf("Should not be able to fill this order: %+v", order)
	}

	po.Path.LimitOpen = 989 // $9.89 option.

	order, err = trdr.constructOrder(po)
	if err != nil {
		t.Errorf("Should be able to fill this order: %+v", order)
	}
}
