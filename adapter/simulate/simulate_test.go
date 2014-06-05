package simulate

import (
	"github.com/eliwjones/thebox/util/interfaces"
	"github.com/eliwjones/thebox/util/structs"

	"testing"
)

func Test_Simulate_Adapter(t *testing.T) {
	var a interfaces.Adapter
	a = New("simulate", "simulation")
	if a == nil {
		t.Errorf("%+v", a)
	}
}

func Test_Simulate_New(t *testing.T) {
	s := New("simulate", "simulator")

	if s.Token == TOKEN {
		t.Errorf("Should not have authed!")
	}

	s = New("simulate", "simulation")

	if s.Token != TOKEN {
		t.Errorf("Should have authed!")
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
	s := New("simulate", "simulation")

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

func Test_Simulate_GetOrders(t *testing.T) {
	s := New("simulate", "simulation")

	o, err := s.GetOrders("open")
	if err != nil {
		t.Errorf("There was an error! err: %s", err)
	}
	if len(o) != 0 {
		t.Errorf("I was not expecting any orders! orders: %+v", o)
	}
	s.SubmitOrder(structs.Order{Symbol: "GOOG"})
	s.SubmitOrder(structs.Order{Symbol: "GOOG"})
	o, err = s.GetOrders("open")
	if err != nil {
		t.Errorf("There was an error! err: %s", err)
	}
	if len(o) != 2 {
		t.Errorf("I was expecting 2 orders! orders: %+v", o)
	}

}

func Test_Simulate_GetPositions(t *testing.T) {
	s := New("simulate", "simulation")

	p, err := s.GetPositions()
	if err != nil {
		t.Errorf("There was an error! err: %s", err)
	}
	if len(p) != 0 {
		t.Errorf("I was not expecting any positions! positions: %+v", p)
	}
}

func Test_Simulate_SubmitOrder(t *testing.T) {
	s := New("simulate", "simulation")
	o := structs.Order{Symbol: "GOOG"}
	orderkey1, err := s.SubmitOrder(o)
	if err != nil {
		t.Errorf("Expected this order submission to succeed! err: %s", err)
	}
	gotOrder, err := s.Get("order", orderkey1)
	if gotOrder != o {
		t.Errorf("Expected: %+v, Got: %+v", o, gotOrder)
	}

	o = structs.Order{Symbol: "AAPL"}
	orderkey2, err := s.SubmitOrder(o)
	o1, _ := s.Get("order", orderkey1)
	o2, _ := s.Get("order", orderkey2)
	if o1 == o2 {
		t.Errorf("%+v should not equal %+v!", o1, o2)
	}
	if o2 != o {
		t.Errorf("Expected: %+v, Got: %+v", o, o2)
	}
}
