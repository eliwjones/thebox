package money

import (
	"github.com/eliwjones/thebox/util/structs"

	"testing"
	"time"
)

func Test_Money_New(t *testing.T) {
	cash := 1000000 * 100
	m := New(cash)
	if m.Total != cash || m.Available != cash {
		t.Errorf("Expected Total: %d, Available: %b to Equal: %d!", m.Total, m.Available, cash)
	}
}

func Test_Money_Get(t *testing.T) {
	m := New(1000000 * 100)
	count := len(m.Allotments)

	a, err := m.Get()
	if err != nil {
		t.Errorf("Why is there an error if there are Allotments!")
	}
	if len(m.Allotments) != count-1 {
		t.Errorf("Expected Len: %d, Got: %d", count-1, len(m.Allotments))
	}
	if m.Available != m.Total-a.Amount {
		t.Errorf("Expected Available: %d, Got: %d", m.Total-a.Amount, m.Available)
	}

	// Empty Allotments and test for err.
	for len(m.Allotments) > 0 {
		m.Get()
	}
	_, err = m.Get()
	if err == nil {
		t.Errorf("Should have received an error since no Allotments are left!")
	}
}

func Test_Money_Put(t *testing.T) {
	m := New(1000000 * 100)
	count := len(m.Allotments)
	total := m.Total

	a := structs.Allotment{Amount: 1000 * 100}
	m.Put(a, true)

	if len(m.Allotments) != count+1 {
		t.Errorf("Expected Len: %d, Got: %d", count+1, len(m.Allotments))
	}

	if m.Total != total+a.Amount {
		t.Errorf("Expected Total: %d, Got: %d", total+a.Amount, m.Total)
	}

	// Verify cannot add empty Allotment.
	count = len(m.Allotments)
	m.Put(structs.Allotment{}, true)
	if len(m.Allotments) != count {
		t.Errorf("Empty Allotment should not have been added!")
	}

}

func Test_Money_ReAllot(t *testing.T) {
	m := New(1000000 * 100)
	count := len(m.Allotments)
	m.Get()
	m.Get()
	if len(m.Allotments) != count-2 {
		t.Errorf("Expected Len: %d, Got: %d", count-2, len(m.Allotments))
	}

	// Not sure if like this.. ReAllot() is really just a wrapper.
	// This is "testing" the underlying reallot() func.
	m.ReAllot()
	if len(m.Allotments) != count {
		t.Errorf("Expected Len: %d, Got: %d", count, len(m.Allotments))
	}
}

func Test_Money_getRandomAllotment(t *testing.T) {
	m := New(1000000 * 100)
	_, err := m.getRandomAllotment()
	if err != nil {
		t.Errorf("Expected random allotment but got err: %s!", err)
	}

	// Verify works with no Allotments.
	for len(m.Allotments) > 0 {
		m.Get()
	}
	_, err = m.getRandomAllotment()
	if err != nil {
		t.Errorf("Expected constructed allotment but got err: %s!", err)
	}

	// Verify works with no Cash and no Allotments.
	m = New(0)
	a, err := m.getRandomAllotment()
	if err == nil {
		t.Errorf("Expected error, but got Allotment: %+v", a)
	}

}

func Test_Money_Processor_Delta(t *testing.T) {
	m := New(1000000 * 100)
	oldTotal := m.Total
	// Send allotments to allotmentIn
	allotment := structs.Allotment{Amount: m.Total / 1000}
	for range 10 {
		m.allotmentIn <- allotment
	}
	for len(m.allotmentIn) > 0 {
		time.Sleep(1 * time.Millisecond)
	}
	if m.Total != oldTotal+allotment.Amount*10 {
		t.Errorf("Expected: %d, Got: %d!", oldTotal+allotment.Amount*10, m.Total)
	}

	// Test for remainder allotment.
	oldTotal = m.Total
	a, _ := m.getRandomAllotment()
	allotment = structs.Allotment{Amount: a.Amount + 200}
	m.allotmentIn <- allotment
	for m.Total == oldTotal {
		time.Sleep(1 * time.Millisecond)
	}
	if m.Total != oldTotal+a.Amount {
		t.Errorf("Expected: %d, Got: %d!", oldTotal+a.Amount, m.Total)
	}
	if m.deltaSum != 200 {
		t.Errorf("Expected: %d, Got: %d!", 200, m.deltaSum)
	}
}

func Test_Money_Dispatcher(t *testing.T) {
	m := New(1000000000)

	ac := make(chan any, 10)
	m.dispatcher.Subscribe("allotment", "tester", ac, true)
	a, _ := m.Get()
	m.dispatcher.Send(a, "allotment")
	gota := <-ac

	if a != gota {
		t.Errorf("Expected: %+v, Got: %+v", a, gota)
	}
}
