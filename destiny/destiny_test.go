package destiny

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"
	"testing"
)

func Test_Destiny_Get(t *testing.T) {
	d := New(1 * 24 * 60 * 60 * 1000)

	if len(d.paths) != 0 {
		t.Errorf("Expected Len: 0, Got: %d!", len(d.paths))
	}
	_, err := d.Get()
	if err == nil {
		t.Errorf("Should have received error.  There are no paths.")
	}

	dest := structs.Destination{Symbol: "GOOG", Type: util.STOCK}
	path := structs.Path{Destination: dest, Timestamp: funcs.MS(funcs.Now())}
	d.paths = append(d.paths, path)
	dp, _ := d.Get()
	if dp != path {
		t.Errorf("Expected: %+v, Got: %+v!", path, dp)
	}
}

func Test_Destiny_Put(t *testing.T) {
	d := New(1 * 24 * 60 * 60 * 1000)

	dest := structs.Destination{Symbol: "GOOG", Type: util.STOCK}
	path := structs.Path{Destination: dest, Timestamp: funcs.MS(funcs.Now())}
	d.Put(path, true)
	if len(d.paths) != 1 {
		t.Errorf("Expected Len: 1, Got: %d!", len(d.paths))
	}
	if d.paths[0] != path {
		t.Errorf("Expected: %+v, Got: %+v!", path, d.paths[0])
	}

	dest = structs.Destination{Symbol: "AAPL", Type: util.STOCK}
	path = structs.Path{Destination: dest, Timestamp: funcs.MS(funcs.Now())}
	d.Put(path, true)
	if len(d.paths) != 2 {
		t.Errorf("Expected Len: 2, Got: %d!", len(d.paths))
	}
	if d.paths[1] != path {
		t.Errorf("Expected: %+v, Got: %+v!", path, d.paths[1])
	}

	// Verify cannot add empty path.
	length := len(d.paths)
	d.Put(structs.Path{}, true)
	if len(d.paths) != length {
		t.Errorf("Empty Destination should not have been added!")
	}
}

func Test_Destiny_Decay(t *testing.T) {
	d := New(1 * 24 * 60 * 60 * 1000)

	now := funcs.MS(funcs.Now())
	dest := structs.Destination{Symbol: "AAPL", Type: util.STOCK}
	path := structs.Path{Destination: dest, Timestamp: now - d.maxage - 10}
	d.paths = append(d.paths, path)

	// Same as with money.ReAllot() testing, this is really just the wrapper.
	// We are testing the underlying decay() func.
	d.Decay()
	if len(d.paths) != 0 {
		t.Errorf("Expected Len: 0, Got: %d!", len(d.paths))
	}
}

func Test_Destiny_New(t *testing.T) {
	d := New(1 * 24 * 60 * 60 * 1000)

	if len(d.paths) != 0 {
		t.Errorf("Expected Len: 0, Got: %d!", len(d.paths))
	}
}
