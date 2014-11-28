package collector

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"

	"testing"
)

func Test_Collector_New(t *testing.T) {
	c := New("/home/mrz/collector", &tdameritrade.TDAmeritrade{})

	if c.root_dir != "/home/mrz/collector" {
		t.Errorf("Expected: %s, Got: %s!", "/home/mrz/collector", c.root_dir)
	}
}

func Test_Collector_Cleanup(t *testing.T) {
	c := New("/home/mrz/collector", &tdameritrade.TDAmeritrade{})

	err := c.Cleanup("20141128")
	if err != nil {
		t.Errorf("Err: %v", err)
	}

	err = c.Cleanup("20140000")
	if err == nil {
		t.Errorf("Expected errors!")
	}
}
