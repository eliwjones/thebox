package pulsar

import (
	"github.com/eliwjones/thebox/util/funcs"

	"fmt"
	"testing"
)

func Test_Pulsar_New(t *testing.T) {
	p := New("data_dir/now", "0000000000", "9999999999")

	if p == nil {
		t.Errorf("Something is broken badly.")
	}

	if len(p.pulses) > 1 {
		t.Errorf("Expected: 1, Got: %d", len(p.pulses))
	}
}

func Test_Pulsar_Pulsing(t *testing.T) {
	p := New("data_dir/all", "2222222222", "5555555555")

	if len(p.pulses) != 4 {
		t.Errorf("Expected: %d, Got: %d", 4, len(p.pulses))
	}

	for i := 0; i < 4; i++ {
		tester := fmt.Sprintf("tester%d", i)
		tc := make(chan int64, 10)
		reply := make(chan int64, 10)
		p.Subscribe(tester, tc, reply)
		go func() {
			for pulse := range tc {
				reply <- pulse
				if pulse == -1 {
					return
				}
			}
		}()
	}

	start := funcs.MS(funcs.Now())
	p.Start()
	finish := funcs.MS(funcs.Now())

	fmt.Printf("MS: %d\n", finish-start)

}
