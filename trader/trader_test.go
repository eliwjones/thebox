package trader

import (
	"github.com/eliwjones/thebox/adapter/simulate"
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"

	"strings"
	"testing"
)

func constructValidStockProtoOrder(td *Trader) structs.ProtoOrder {
	minCommission := td.commission[util.STOCK]["base"] + td.commission[util.STOCK]["unit"]

	po := structs.ProtoOrder{Allotment: structs.Allotment{}, Path: structs.Path{}}
	po.Path.Destination.Type = util.STOCK
	po.Path.Destination.Symbol = "GOOG"

	po.Path.LimitOpen = 1000
	po.Allotment.Amount = po.Path.LimitOpen*td.multiplier[util.STOCK] + minCommission
	return po
}

func constructValidOptionProtoOrder(td *Trader) structs.ProtoOrder {
	minCommission := td.commission[util.OPTION]["base"] + td.commission[util.OPTION]["unit"]

	po := structs.ProtoOrder{Allotment: structs.Allotment{}, Path: structs.Path{}}
	po.Path.Destination.Type = util.OPTION
	po.Path.Destination.Symbol = "GOOG MAY 2014 1234 PUT"

	po.Path.LimitOpen = 1000
	po.Allotment.Amount = po.Path.LimitOpen*td.multiplier[util.OPTION] + minCommission
	return po
}

func Test_Trader_New(t *testing.T) {
	td := New(simulate.New("simulate", "simulation"))
	if td == nil {
		t.Errorf("Trader New() call failed!")
	}
}

func Test_Trader_constructOrder_Option(t *testing.T) {
	td := New(simulate.New("simulate", "simulation"))

	po := constructValidOptionProtoOrder(td)

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
	td := New(simulate.New("simulate", "simulation"))

	po := constructValidStockProtoOrder(td)

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

func Test_Trader_Processor_ProtoOrder(t *testing.T) {
	td := New(simulate.New("simulate", "simulation"))

	po := constructValidStockProtoOrder(td)

	reply := make(chan interface{})

	td.pomIn <- structs.ProtoOrderMessage{ProtoOrder: po, Reply: reply}
	response := <-reply
	if !strings.HasPrefix(response.(string), "order-") {
		t.Errorf("Expected: order-*, Got: %s!", response.(string))
	}

	// Invalid ProtoOrder should be sent back.
	po.Path.LimitOpen++
	td.pomIn <- structs.ProtoOrderMessage{ProtoOrder: po, Reply: reply}
	response = <-reply
	if response != po {
		t.Errorf("Expected: %+v, Got: %+v!", po, response)
	}
}
