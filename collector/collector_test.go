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
