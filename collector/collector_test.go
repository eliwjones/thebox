package collector

import (
	"testing"
)

func Test_Collector_New(t *testing.T) {
	c := New("../cmd/collectord")

	if c.root_dir != "../cmd/collectord" {
		t.Errorf("Expected: %s, Got: %s!", "../cmd/collectord", c.root_dir)
	}
}

func Test_Collector_Cleanup(t *testing.T) {
	// Sub-optimal, but just looking above and over in collectord dir for a 'data' directory.
	c := New("../cmd/collectord")

	err := c.Clean("20141201")
	if err != nil {
		t.Errorf("Err: %v", err)
	}

	err = c.Clean("20140000")
	if err == nil {
		t.Errorf("Expected errors!")
	}
}

func Test_Collector_isNear(t *testing.T) {
	// EST Testing.
	time1 := "204801"
	time2 := "154801"
	near, diff := isNear(time1, time2)
	if !near {
		t.Errorf("Expected near result for: %s, %s", time1, time2)
	}
	if diff != 0 {
		t.Errorf("Expected 0, Got: %d", int(diff))
	}

	time1 = "204801"
	time2 = "154851"
	near, diff = isNear(time1, time2)
	if near {
		t.Errorf("Did not expect near result for: %s, %s", time1, time2)
	}
	if diff != 50 {
		t.Errorf("Expected 50, Got: %d", int(diff))
	}

	// EDT Testing
	time1 = "204801"
	time2 = "164821"
	near, diff = isNear(time1, time2)
	if !near {
		t.Errorf("Expected near result for: %s, %s", time1, time2)
	}
	if diff != 20 {
		t.Errorf("Expected 20, Got: %d", int(diff))
	}

	time1 = "204801"
	time2 = "164851"
	near, diff = isNear(time1, time2)
	if near {
		t.Errorf("Did not expect near result for: %s, %s", time1, time2)
	}
	if diff != 50 {
		t.Errorf("Expected 50, Got: %d", int(diff))
	}
}
