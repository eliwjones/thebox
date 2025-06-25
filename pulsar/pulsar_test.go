package pulsar

import (
	"github.com/eliwjones/thebox/util/funcs"

	"fmt"
	"testing"
)

func Test_Pulsar_New(t *testing.T) {
	p := New("data_dir/now", "0000000000", "9999999999", false)

	if p == nil {
		t.Errorf("Something is broken badly.")
	}

	if len(p.pulses) > 1 {
		t.Errorf("Expected: 1, Got: %d", len(p.pulses))
	}
}

func Test_Pulsar_Pulsing(t *testing.T) {
	p := New("data_dir/all", "2222222222", "5555555555", true)

	if len(p.pulses) != 4 {
		t.Errorf("Expected: %d, Got: %d", 4, len(p.pulses))
	}

	for i := range 4 {
		tester := fmt.Sprintf("tester%d", i)
		tc := make(chan int64, 10)
		reply := make(chan int64, 10)
		p.Subscribe(tester, tc, reply)
		go func() {
			lastPulse := int64(0)
			for pulse := range tc {
				reply <- pulse
				if pulse == -1 {
					return
				}
				if !(pulse > lastPulse) {
					t.Errorf("Expecting this pulse: %d to be greater than lastPulse: %d", pulse, lastPulse)
				}
				lastPulse = pulse
			}
		}()
	}

	start := funcs.MS(funcs.Now())
	p.Start()
	finish := funcs.MS(funcs.Now())

	fmt.Printf("MS: %d\n", finish-start)

}
