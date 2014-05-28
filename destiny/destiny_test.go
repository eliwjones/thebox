package destiny

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"testing"
	"time"
)

func Test_Destiny_New(t *testing.T) {
	d := New(1 * 24 * 60 * 60 * 1000)

	if len(d.paths) != 0 {
		t.Errorf("Expected Len: 0, Got: %d!", len(d.paths))
	}
}

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

func Test_Destiny_Processor_Allotment(t *testing.T) {
	d := New(1 * 24 * 60 * 60 * 1000)
	tc := make(chan interface{}, 10)
	d.dispatcher.Subscribe("protoorder", "trader", tc, true)

	p := structs.Path{LimitClose: 1, LimitOpen: 2, Timestamp: 3}
	d.Put(p, true)

	a := structs.Allotment{Amount: 100}
	reply := make(chan interface{})
	d.amIn <- structs.AllotmentMessage{Allotment: a, Reply: reply}

	response := <-reply
	if !response.(bool) {
		t.Errorf("Should have received 'true'")
	}
	if len(tc) != 1 {
		t.Errorf("Expected: 1, Got: %d!", len(tc))
	}
	po := <-tc
	if (po != structs.ProtoOrder{Allotment: a, Path: p}) {
		t.Errorf("Expected: %+v, Got: %+v!", structs.ProtoOrder{Allotment: a, Path: p}, po)
	}

	// Verify kickback of Allotment when no Paths.
	d = New(1 * 24 * 60 * 60 * 1000)
	d.amIn <- structs.AllotmentMessage{Allotment: a, Reply: reply}
	response = <-reply
	if response != a {
		t.Errorf("No Path. Expected Allotment: %+v, Got: %+v", a, response)
	}
}

func Test_Destiny_Processor_Delta(t *testing.T) {
	// Currently using dummy heuristic so just test that.
	d := New(1 * 24 * 60 * 60 * 1000)
	pathLen := len(d.paths)

	delta := structs.Delta{}
	delta.Path = structs.Path{LimitClose: 1, LimitOpen: 2, Timestamp: 3}
	delta.Percent = 99

	d.dIn <- delta
	for len(d.dIn) > 0 {
		time.Sleep(1 * time.Millisecond)
	}
	if pathLen != len(d.paths) {
		t.Errorf("Expected: %d, Got: %d!", pathLen, len(d.paths))
	}

	// Passing dummy heuristic threshold, should see extra path.
	delta.Percent = 101

	d.dIn <- delta
	for len(d.dIn) > 0 {
		time.Sleep(1 * time.Millisecond)
	}
	if pathLen+1 != len(d.paths) {
		t.Errorf("Expected: %d, Got: %d!", pathLen+1, len(d.paths))
	}
}
