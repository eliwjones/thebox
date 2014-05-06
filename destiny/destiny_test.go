package destiny

import (
	"github.com/eliwjones/thebox/trader"
	"github.com/eliwjones/thebox/util"
	"testing"
)

func Test_Destiny(t *testing.T) {
	onedayinmilliseconds := int64(1 * 24 * 60 * 60 * 1000)
	destiny := New(onedayinmilliseconds)

	if len(destiny.paths) != 0 {
		t.Errorf("Should be 0 destinations, but there are %d!", len(destiny.paths))
	}

	path, err := destiny.Get()
	if err == nil {
		t.Errorf("Should have received error, but got path: %v, err: %s", path, err)
	}

	destination := Destination{Symbol: "GOOG", Type: trader.STOCK}
	path = Path{Destination: destination, Timestamp: util.MS(util.Now())}
	destiny.Put(path, true)
	if len(destiny.paths) != 1 {
		t.Errorf("Should be 1 path, but there are %d!", len(destiny.paths))
	}

	dp, _ := destiny.Get()
	if dp != path {
		t.Errorf("Expected: %+v, Got: %+v!", path, dp)
	}

	now := util.MS(util.Now())
	destination = Destination{Symbol: "AAPL", Type: trader.STOCK}
	path = Path{Destination: destination, Timestamp: now - destiny.maxage - 10}
	destiny.Put(path, true)
	if len(destiny.paths) != 2 {
		t.Errorf("Should be 2 paths, but there are %d!", len(destiny.paths))
	}

	destiny.Decay()
	if len(destiny.paths) != 1 {
		t.Errorf("Should be 1 path now, but there are %d!", len(destiny.paths))
	}

	// Verify cannot add empty path.
	length := len(destiny.paths)
	destiny.Put(Path{}, true)
	if len(destiny.paths) != length {
		t.Errorf("Empty Destination should not have been added!")
	}
}
