package money

import (
	"testing"
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
