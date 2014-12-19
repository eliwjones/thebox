package collector

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/structs"

	"fmt"
	"os"
	"testing"
)

func Test_Collector_New(t *testing.T) {
	c := New("./testdir")

	if c.rootdir != "./testdir" {
		t.Errorf("Expected: %s, Got: %s!", "../cmd/collectord", c.rootdir)
	}
}

func Test_Collector_collect(t *testing.T) {
	c := New("./testdir")

	// Suppose may need some sort of LoadAdapterFromConfig() function somewhere.
	lines, _ := funcs.GetConfig(c.rootdir + "/config")
	id := lines[0]
	pass := lines[1]
	sid := lines[2]
	jsess := lines[3]

	tda := tdameritrade.New(id, pass, sid, jsess)
	if tda.JsessionID != jsess {
		lines[3] = tda.JsessionID
		funcs.UpdateConfig(c.rootdir+"/config", lines)
	}
	c.Adapter = tda
	symbol := "INTC"
	thisMonth, limitMonth := c.collect(symbol)

	select {
	case reply := <-c.replies:
		r, _ := reply.(bool)
		if !r {
			t.Errorf("Received False Reply from c.replies")
		}
	case reply := <-c.pipe:
		s, success := reply.Data.(structs.Stock)
		if !success {
			t.Errorf("Expected to get Stock back first.")
		}
		if s.Symbol != symbol {
			t.Errorf("Expected: %s, Got: %s!", symbol, s.Symbol)
		}
	}

	if len(c.pipe) == 0 {
		t.Errorf("Expecting Non-zero Pipe.")
	}

	seenThisMonth := false
	seenLimitMonth := false
	// If here, should read out all options.
	for message := range c.pipe {
		if message.Data == nil {
			break
		}
		o, success := message.Data.(structs.Option)
		if !success {
			t.Errorf("Expecting an Option here!")
		}
		if o.Expiration[:6] != thisMonth && o.Expiration[:6] != limitMonth {
			t.Errorf("Expected Expiration %s or %s, Got: %s", thisMonth, limitMonth, o.Expiration)
		}
		if o.Expiration[:6] == thisMonth {
			seenThisMonth = true
		}
		if o.Expiration[:6] == limitMonth {
			seenLimitMonth = true
		}
	}

	if !seenThisMonth {
		t.Errorf("Did not see Exp: %s", thisMonth)
	}
	if !seenLimitMonth {
		t.Errorf("Did not see Exp: %s", seenLimitMonth)
	}
}

func Test_Collector_dumpTargets(t *testing.T) {
	c := New("./testdir")

	c.targets = map[string]map[string]target{"current": map[string]target{}, "next": map[string]target{}}
	c.targets["current"]["AAPL"] = target{Timestamp: int64(1234567890)}
	c.targets["current"]["BABA"] = target{Timestamp: int64(1234567890)}

	c.dumpTargets()
}

func Test_Collector_loadTargets(t *testing.T) {
	c := New("./testdir")

	targets := c.loadTargets()

	if targets == nil {
		t.Errorf("Expected, at least, empty targets!")
	}
	if len(targets["current"]) != 2 {
		t.Errorf("Expecting 2 'current' targets.")
	}
	if len(targets["next"]) > 0 {
		t.Errorf("Not expecting 'next' targets.")
	}

	os.RemoveAll("./testdir/live")
}

func Test_Collector_maybeCycleTargets(t *testing.T) {
	c := New("./testdir")
	start_ts := int64(10 * 60)
	next_ts := start_ts + int64(10*60)
	c.targets["current"]["GOOG"] = target{Timestamp: start_ts}
	c.targets["next"]["GOOG"] = target{Timestamp: next_ts}

	// Should silently pass over old timestamp.
	ts := c.targets["current"]["GOOG"].Timestamp - 50
	c.maybeCycleTargets(ts)
	if c.targets["current"]["GOOG"].Timestamp != start_ts {
		t.Errorf("Current Target Timestamp should not have advanced!")
	}
	if c.targets["next"]["GOOG"].Timestamp != next_ts {
		t.Errorf("Next Target Timestamp should not have advanced!")
	}

	// Should cycle for timestamp past 5 minute midpoint.
	ts = c.targets["current"]["GOOG"].Timestamp + int64(5*60)
	c.maybeCycleTargets(ts)
	if c.targets["current"]["GOOG"].Timestamp != next_ts {
		t.Errorf("Target Timestamp Expected: %d, Got: %d", next_ts, c.targets["current"]["GOOG"].Timestamp)
	}
	if c.targets["next"]["GOOG"].Timestamp == next_ts {
		t.Errorf("Next Target Timestamp should have advanced!")
	}
}

func Test_Collector_isNear(t *testing.T) {
	padding := 45

	// EST Testing.
	time1 := funcs.ClockTimeInSeconds("204801")
	time2 := funcs.ClockTimeInSeconds("154801")
	near, diff := isNear(time1, time2, padding)
	if !near {
		t.Errorf("Expected near result for: %s, %s", time1, time2)
	}
	if diff != 0 {
		t.Errorf("Expected 0, Got: %d", int(diff))
	}

	time1 = funcs.ClockTimeInSeconds("204801")
	time2 = funcs.ClockTimeInSeconds("154851")
	near, diff = isNear(time1, time2, padding)
	if near {
		t.Errorf("Did not expect near result for: %s, %s", time1, time2)
	}
	if diff != 50 {
		t.Errorf("Expected 50, Got: %d", int(diff))
	}

	// EDT Testing
	time1 = funcs.ClockTimeInSeconds("204801")
	time2 = funcs.ClockTimeInSeconds("164821")
	near, diff = isNear(time1, time2, padding)
	if !near {
		t.Errorf("Expected near result for: %s, %s", time1, time2)
	}
	if diff != 20 {
		t.Errorf("Expected 20, Got: %d", int(diff))
	}

	time1 = funcs.ClockTimeInSeconds("204801")
	time2 = funcs.ClockTimeInSeconds("164851")
	near, diff = isNear(time1, time2, padding)
	if near {
		t.Errorf("Did not expect near result for: %s, %s", time1, time2)
	}
	if diff != 50 {
		t.Errorf("Expected 50, Got: %d", int(diff))
	}
}

func Test_Collector_logError(t *testing.T) {
	os.RemoveAll("./testdir/error")
	c := New("./testdir")
	c.logError("testfunc", fmt.Errorf("Test error of type 'error'"))
	c.logError("testfunc", "Test error of type 'string'")
}
