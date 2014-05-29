package pulsar

import (
	"testing"
	"time"
)

func Test_Pulsar_New(t *testing.T) {
	p := New(1 * time.Millisecond)

	if p == nil {
		t.Errorf("Something is broken badly.")
	}
}

func Test_Pulsar_Period(t *testing.T) {
	period := 1 * time.Millisecond
	p := New(period)

	timer1 := make(chan interface{}, 200)
	p.Subscribe("tester1", timer1)

	timer2 := make(chan interface{}, 200)
	p.Subscribe("tester2", timer2)

	count := 100
	time.Sleep(time.Duration(count) * period)
	if len(timer1) != count {
		t.Errorf("Expected: %d, Got: %d!", count, len(timer1))
	}
	if len(timer2) != count {
		t.Errorf("Expected: %d, Got: %d!", count, len(timer2))
	}
}
