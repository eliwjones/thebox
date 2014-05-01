package money

import (
	"testing"
)

func Test_Money(t *testing.T) {
	cash := 1000000 * 100 // Million $ in cents.
	money := NewMoney(cash)
	money.ReAllot()

	total := len(money.Allotments)
	allotment, err := money.Get()

	if err != nil {
		t.Errorf("Why is there an error if there are Allotments!")
	}
	if len(money.Allotments) != total-1 {
		t.Errorf("Len of Allotments should be %d, but got: %d", total-1, len(money.Allotments))
	}

	if money.Available != money.Total-allotment.Amount {
		t.Errorf("Should be less Available than Total.  Available: %d", money.Available)
	}

	money.Put(allotment, true)

	if money.Available != money.Total {
		t.Errorf("Available money should equal Total money! Available: %d", money.Available)
	}

	// Empty Allotments and test for err.
	for len(money.Allotments) > 0 {
		money.Get()
	}
	allotment, err = money.Get()
	if err == nil {
		t.Errorf("Should have received an error since no Allotments are left!")
	}

	// Verify cannot add empty Allotment.
	length := len(money.Allotments)
	money.Put(Allotment{}, true)
	if len(money.Allotments) != length {
		t.Errorf("Empty Allotment should not have been added!")
	}
}
