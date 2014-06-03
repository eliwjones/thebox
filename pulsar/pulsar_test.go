package pulsar

import (
	"github.com/eliwjones/thebox/util/funcs"

	"testing"
	"time"
)

func Test_Pulsar_New(t *testing.T) {
	p := New(funcs.MS(funcs.Now()), 1)

	if p == nil {
		t.Errorf("Something is broken badly.")
	}
}

func Test_Pulsar_Periods(t *testing.T) {
	for period := range periods {
		p := New(0, period)
		count := period / 4

		timer1 := make(chan interface{}, 2*count)
		p.Subscribe("tester1", timer1)

		timer2 := make(chan interface{}, 2*count)
		p.Subscribe("tester2", timer2)

		now := p.now
		time.Sleep(time.Duration(count) * p.period)
		if len(timer1) < count*98/100 {
			t.Errorf("Period: %d - Expected: %d, Got: %d!", period, count, len(timer1))
		}
		if len(timer2) < count*98/100 {
			t.Errorf("Period: %d - Expected: %d, Got: %d!", period, count, len(timer2))
		}
		if p.now < int64(count*1000*98/100)+now {
			t.Errorf("Period: %d - Expected: %d, Got: %d! Diff: %d.", period, int64(count*1000)+now, p.now, (int64(count*1000)+now)-p.now)
		}
	}
}
