package simulate

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"

	"testing"
)

func Test_Simulate_Adapter(t *testing.T) {
	a := New("simulate", "simulation", 300000*100)
	if a == nil {
		t.Errorf("%+v", a)
	}
}

func Test_Simulate_New(t *testing.T) {
	s := New("simulate", "simulator", 300000*100)

	if s.Token == TOKEN {
		t.Errorf("Should not have authed!")
	}

	s = New("simulate", "simulation", 300000*100)

	if s.Token != TOKEN {
		t.Errorf("Should have authed!")
	}
}

func Test_Simulate_ClosePosition(t *testing.T) {
	s := New("simulate", "simulation", 300000*100)
	startCash := s.Cash
	oid, _ := s.SubmitOrder(structs.Order{Symbol: "GOOG_OPTION", Type: util.OPTION, Volume: 100, Limitprice: 300})
	if !(s.Cash < startCash) {
		t.Errorf("I don't appear to be decrementing Cash.")
	}
	if s.Cash == s.Value {
		t.Errorf("Cash should NOT equal Value! %d == %d", s.Cash, s.Value)
	}
	s.ClosePosition(oid, 600)
	if !(s.Cash > startCash) {
		t.Errorf("Expected to have more Cash now!")
	}
	if s.Cash != s.Value {
		t.Errorf("Cash should equal Value! %d != %d", s.Cash, s.Value)
	}
}

func Test_Simulate_Connect(t *testing.T) {
	s := &Simulate{}

	token, _ := s.Connect("simulate", "simulation", "")
	if token != TOKEN {
		t.Errorf("Should have received TOKEN!")
	}
}

func Test_Simulate_Get(t *testing.T) {
	s := New("simulate", "simulation", 300000*100)

	_, err := s.Get("thang", "thing")
	if err == nil {
		t.Errorf("Expected error for non-existent 'thang'.")
	}

	cash, err := s.Get("cash", "")
	if err != nil {
		t.Errorf("Should have gotten cash! cash: %d, err: %s", cash, err)
	}
	if cash != s.Cash {
		t.Errorf("Expected: %d, Got: %d!", s.Cash, cash)
	}

	value, err := s.Get("value", "")
	if err != nil {
		t.Errorf("Should have gotten value! value: %d, err: %s", value, err)
	}
	if value != s.Value {
		t.Errorf("Expected: %d, Got: %d!", s.Value, value)
	}

	order, err := s.Get("order", "non-existent-key")
	if err == nil {
		t.Errorf("Should have received error for 'non-existent-key'! order: %+v", order)
	}
	s.Orders["existing-key"] = structs.Order{}
	order, err = s.Get("order", "existing-key")
	if err != nil {
		t.Errorf("Got error but order should exist!")
	}
	if order != s.Orders["existing-key"] {
		t.Errorf("Expected: %+v, Got: %+v", s.Orders["existing-key"], order)
	}

	position, err := s.Get("position", "non-existent-key")
	if err == nil {
		t.Errorf("Should have received error for 'non-existent-key'! position: %+v", position)
	}
	s.Positions["existing-key"] = structs.Position{}
	position, err = s.Get("position", "existing-key")
	if err != nil {
		t.Errorf("Got error but position should exist!")
	}
	if position != s.Positions["existing-key"] {
		t.Errorf("Expected: %+v, Got: %+v", s.Positions["existing-key"], position)
	}
}

func Test_Simulate_GetBalances(t *testing.T) {
	s := New("simulate", "simulation", 300000*100)
	b, err := s.GetBalances()
	if err != nil {
		t.Errorf("Err: %s", err)
	}
	if b["cash"] == 0 {
		t.Errorf("Expected non-zero cash. Got: %d", b["cash"])
	}
	if b["value"] == 0 {
		t.Errorf("Expected non-zero value. Got: %d", b["value"])
	}
}

func Test_Simulate_GetPositions(t *testing.T) {
	s := New("simulate", "simulation", 300000*100)

	p, err := s.GetPositions()
	if err != nil {
		t.Errorf("There was an error! err: %s", err)
	}
	if len(p) != 0 {
		t.Errorf("I was not expecting any positions! positions: %+v", p)
	}
	s.SubmitOrder(structs.Order{Symbol: "GOOG_OPTION", Type: util.OPTION, Volume: 100, Limitprice: 300})
	s.SubmitOrder(structs.Order{Symbol: "GOOG_OPTION", Type: util.OPTION, Volume: 100, Limitprice: 300})
	// Submitted orders instantly turn into positions.
	p, err = s.GetPositions()
	if err != nil {
		t.Errorf("There was an error! err: %s", err)
	}
	if len(p) != 2 {
		t.Errorf("I was expecting 2 positions! positions: %+v", p)
	}

}

func Test_Simulate_SubmitOrder(t *testing.T) {
	s := New("simulate", "simulation", 300000*100)
	o := structs.Order{Symbol: "GOOG_OPTION", Type: util.OPTION, Volume: 100, Limitprice: 300}
	orderkey1, err := s.SubmitOrder(o)
	o.Id = orderkey1
	if err != nil {
		t.Errorf("Expected this order submission to succeed! err: %s", err)
	}
	if s.Positions[orderkey1].Order != o {
		t.Errorf("Expected order to turn into Position!\n%v\n%v", o, s.Positions[orderkey1].Order)
	}
}
