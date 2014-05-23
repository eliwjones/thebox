package trader

import (
	"github.com/eliwjones/thebox/money"
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"
	"testing"
)

func Test_Trader_New(t *testing.T) {
	td := New(10)
	if td == nil {
		t.Errorf("Trader New() call failed!")
	}
}

func Test_Trader_constructOrder_Option(t *testing.T) {
	td := New(10)

	minCommission := td.commission[util.OPTION]["base"] + td.commission[util.OPTION]["unit"]

	po := ProtoOrder{Allotment: money.Allotment{}, Path: structs.Path{}}
	po.Path.Destination.Type = util.OPTION
	po.Path.Destination.Symbol = "GOOG MAY 2014 1234 PUT"

	po.Path.LimitOpen = 1000
	po.Allotment.Amount = po.Path.LimitOpen*td.multiplier[util.OPTION] + minCommission

	o, err := td.constructOrder(po)
	if err != nil {
		t.Errorf("Should be able to fill this order: %+v, Allotment: %d", o, po.Allotment.Amount)
	}

	po.Path.LimitOpen++
	o, err = td.constructOrder(po)
	if err == nil {
		t.Errorf("Should not be able to fill this order: %+v", o)
	}
}

func Test_Trader_constructOrder_Stock(t *testing.T) {
	td := New(10)

	minCommission := td.commission[util.STOCK]["base"] + td.commission[util.STOCK]["unit"]

	po := ProtoOrder{Allotment: money.Allotment{}, Path: structs.Path{}}
	po.Path.Destination.Type = util.STOCK
	po.Path.Destination.Symbol = "GOOG"

	po.Path.LimitOpen = 1000
	po.Allotment.Amount = po.Path.LimitOpen*td.multiplier[util.STOCK] + minCommission

	o, err := td.constructOrder(po)
	if err != nil {
		t.Errorf("Should be able to fill this order: %+v", o)
	}

	po.Path.LimitOpen++
	o, err = td.constructOrder(po)
	if err == nil {
		t.Errorf("Should not be able to fill this order: %+v", o)
	}

}
