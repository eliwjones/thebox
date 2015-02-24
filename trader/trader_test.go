package trader

import (
	"github.com/eliwjones/thebox/adapter/simulate"
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"

	"reflect"
	"strings"
	"testing"
)

func constructValidStockProtoOrder(td *Trader) structs.ProtoOrder {
	po := structs.ProtoOrder{}
	po.Type = util.STOCK
	po.Symbol = "GOOG"
	po.LimitOpen = 1000

	return po
}

func constructValidOptionProtoOrder(td *Trader) structs.ProtoOrder {
	po := structs.ProtoOrder{}
	po.Type = util.OPTION
	po.Symbol = "GOOG MAY 2014 1234 PUT"
	po.LimitOpen = 1000

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
	minCommission := td.commission[util.OPTION]["base"] + td.commission[util.OPTION]["unit"]
	allotment := po.LimitOpen*td.multiplier[util.OPTION] + minCommission

	o, err := td.constructOrder(po, allotment)
	if err != nil {
		t.Errorf("Should be able to fill this order: %+v, Allotment: %d", o, td.Allotments[0])
	}

	po.LimitOpen++
	o, err = td.constructOrder(po, allotment)
	if err == nil {
		t.Errorf("Should not be able to fill this order: %+v", o)
	}
}

func Test_Trader_constructOrder_Stock(t *testing.T) {
	td := New(simulate.New("simulate", "simulation"))

	po := constructValidStockProtoOrder(td)
	minCommission := td.commission[util.STOCK]["base"] + td.commission[util.STOCK]["unit"]
	allotment := po.LimitOpen*td.multiplier[util.STOCK] + minCommission

	o, err := td.constructOrder(po, allotment)
	if err != nil {
		t.Errorf("Should be able to fill this order: %+v", o)
	}

	po.LimitOpen++
	o, err = td.constructOrder(po, allotment)
	if err == nil {
		t.Errorf("Should not be able to fill this order: %+v", o)
	}

}

func Test_Trader_Processor_ProtoOrder(t *testing.T) {
	td := New(simulate.New("simulate", "simulation"))

	po := constructValidStockProtoOrder(td)
	minCommission := td.commission[util.STOCK]["base"] + td.commission[util.STOCK]["unit"]
	td.Allotments = []int{po.LimitOpen*td.multiplier[util.STOCK] + minCommission}

	reply := make(chan interface{})

	po.Reply = reply
	td.PoIn <- po

	// Send pulse so will do something.
	td.Pulses <- int64(1)

	response := <-reply
	if !strings.HasPrefix(response.(string), "order-") {
		t.Errorf("Expected: order-*, Got: %s!", response.(string))
	}

	// Invalid ProtoOrder should be sent back.
	td.Allotments = []int{po.LimitOpen*td.multiplier[util.STOCK] + minCommission}
	po.LimitOpen++
	td.PoIn <- po
	td.Pulses <- int64(1)
	response = <-reply
	if response != po {
		t.Errorf("Expected: %+v, Got: %+v!", po, response)
	}
}

func Test_Trader_serializeState_deserializeState(t *testing.T) {
	td := New(simulate.New("simulate", "simulation"))

	td.CurrentWeekId = int64(9999)
	td.Allotments = allotments()

	po := structs.ProtoOrder{Symbol: "GOOG_201501_p", Type: util.OPTION, LimitOpen: 100, Timestamp: int64(1)}
	o, err := td.constructOrder(po, td.Allotments[0])
	if err != nil {
		t.Errorf("Expected Order.  Err: %s", err)
	}
	td.Positions = map[string]structs.Position{}
	td.Positions["order-1"] = structs.Position{Id: "order-1", Fillprice: o.Limitprice, Order: o}

	st, err := td.serializeState()
	if err != nil {
		t.Errorf("%s", err)
	}
	newTrader := New(simulate.New("simulate", "simulation"))

	err = newTrader.deserializeState(st)
	if err != nil {
		t.Errorf("Err: %s, dt: %v", err, newTrader)
	}

	// Verify newTrader received state.
	if newTrader.CurrentWeekId != td.CurrentWeekId {
		t.Errorf("Expected: %d, Got: %d", td.CurrentWeekId, newTrader.CurrentWeekId)
	}
	if !reflect.DeepEqual(newTrader.Allotments, td.Allotments) {
		t.Errorf("Expected: %v, Got: %v", td.Allotments, newTrader.Allotments)
	}
	if !reflect.DeepEqual(newTrader.Positions, td.Positions) {
		t.Errorf("\nExpected: %v\nGot: %v", td.Positions, newTrader.Positions)
	}
}
