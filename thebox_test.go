package thebox

import (
	"testing"
)

func Test_Money(t *testing.T) {
	cash := 1000000 * 100 // Million $ in cents.
	money := NewMoney(cash)
	money.ReAllot()

	total := len(money.Allotments)
	allotment := money.Get()

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
}
