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

	a := Allotment{Amount: 1000 * 100}
	m.Put(a, true)

	if len(m.Allotments) != count+1 {
		t.Errorf("Expected Len: %d, Got: %d", count+1, len(m.Allotments))
	}

	if m.Total != total+a.Amount {
		t.Errorf("Expected Total: %d, Got: %d", total+a.Amount, m.Total)
	}

	// Verify cannot add empty Allotment.
	count = len(m.Allotments)
	m.Put(Allotment{}, true)
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
	a, err := m.getRandomAllotment()
	if err != nil {
		t.Errorf("Expected random allotment but got err: %s!", err)
	}

	// Verify works with no Allotments.
	for len(m.Allotments) > 0 {
		m.Get()
	}
	a, err = m.getRandomAllotment()
	if err != nil {
		t.Errorf("Expected constructed allotment but got err: %s!", err)
	}

	// Verify works with no Cash and no Allotments.
	m = New(0)
	a, err = m.getRandomAllotment()
	if err == nil {
		t.Errorf("Expected error, but got Allotment: %+v", a)
	}

}

func Test_Money_Processor_Delta(t *testing.T) {
	m := New(1000000 * 100)
	oldTotal := m.Total
	// Send deltas to deltaIn
	d := structs.Delta{Amount: m.Total / 1000}
	for i := 0; i < 10; i++ {
		m.deltaIn <- d
	}
	for len(m.deltaIn) > 0 {
		time.Sleep(1 * time.Millisecond)
	}
	if m.Total != oldTotal+d.Amount*10 {
		t.Errorf("Expected: %d, Got: %d!", oldTotal+d.Amount*10, m.Total)
	}

	// Test for remainder delta.
	oldTotal = m.Total
	a, _ := m.getRandomAllotment()
	d = structs.Delta{Amount: a.Amount + 200}
	m.deltaIn <- d
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
