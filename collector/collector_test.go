package collector

import (
	"github.com/eliwjones/thebox/util/funcs"

	"os"
	"testing"
)

func Test_Collector_New(t *testing.T) {
	c := New("../cmd/collectord")

	if c.rootdir != "../cmd/collectord" {
		t.Errorf("Expected: %s, Got: %s!", "../cmd/collectord", c.rootdir)
	}
}

func Test_Collector_dumpTargets(t *testing.T) {
	c := New(".")

	c.targets = map[string]map[string]target{"current": map[string]target{}, "next": map[string]target{}}
	c.targets["current"]["AAPL"] = target{Timestamp: int64(1234567890)}
	c.targets["current"]["BABA"] = target{Timestamp: int64(1234567890)}

	c.dumpTargets()
}

func Test_Collector_loadTargets(t *testing.T) {
	c := New(".")

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

	os.RemoveAll("./live")
}

func Test_Collector_maybeCycleTargets(t *testing.T) {
	c := New(".")
	start_ts := int64(100)
	next_ts := start_ts + int64(10*60)
	c.targets["current"]["GOOG"] = target{Timestamp: start_ts}
	c.targets["next"]["GOOG"] = target{Timestamp: next_ts}

	ts := c.targets["current"]["GOOG"].Timestamp - 50
	c.maybeCycleTargets(ts)
	if c.targets["current"]["GOOG"].Timestamp != start_ts {
		t.Errorf("Current Target Timestamp should not have advanced!")
	}
	if c.targets["next"]["GOOG"].Timestamp != next_ts {
		t.Errorf("Next Target Timestamp should not have advanced!")
	}

	ts = c.targets["current"]["GOOG"].Timestamp + 50
	c.maybeCycleTargets(ts)
	if c.targets["current"]["GOOG"].Timestamp != next_ts {
		t.Errorf("Target Timestamp should have advanced!")
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
