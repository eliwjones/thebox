package funcs

import (
	"github.com/eliwjones/thebox/util/structs"

	"math"
	"reflect"
	"testing"
	"time"
)

func Test_ChooseMFromN(t *testing.T) {
	j := 5
	for i := 0; i < j+2; i++ {
		bag := ChooseMFromN(i, j)
		l := int(math.Min(float64(i), float64(j)))
		if len(bag) != l {
			t.Errorf("Expected bag of length %d! Got %d!", l, len(bag))
		}

	}
}

func Test_EncodeDecodeMaximum(t *testing.T) {
	m := structs.Maximum{Expiration: "20150101", OptionSymbol: "GOOG_013015C610", Timestamp: int64(1000000001),
		Underlying: "GOOG", MaximumBid: 100, OptionAsk: 50, OptionBid: 40, OptionType: "c", Strike: 610, UnderlyingBid: 50000, Volume: 100}
	m.MaxTimestamp = m.Timestamp + int64(24*60*60)

	em, err := Encode(&m, MaximumEncodingOrder)
	if err != nil {
		t.Errorf("Encode Maximum err: %s!", err)
	}

	m2 := structs.Maximum{}
	err = Decode(em, &m2, MaximumEncodingOrder)
	if err != nil {
		t.Errorf("Decode Maximum err: %s!", err)
	}

	// Not sure why, but feel like spelling this out here.
	if m2.Timestamp != m.Timestamp {
		t.Errorf("Expected: %d, Got: %d", m.Timestamp, m2.Timestamp)
	}
	if m2.OptionType != m.OptionType {
		t.Errorf("Expected: %s, Got: %s", m.OptionType, m2.OptionType)
	}
	if m2.Underlying != m.Underlying {
		t.Errorf("Expected: %s, Got: %s", m.Underlying, m2.Underlying)
	}
	if m2.OptionSymbol != m.OptionSymbol {
		t.Errorf("Expected: %s, Got: %s", m.OptionSymbol, m2.OptionSymbol)
	}
	if m2.UnderlyingBid != m.UnderlyingBid {
		t.Errorf("Expected: %d, Got: %d", m.UnderlyingBid, m2.UnderlyingBid)
	}
	if m2.OptionAsk != m.OptionAsk {
		t.Errorf("Expected: %d, Got: %d", m.OptionAsk, m2.OptionAsk)
	}
	if m2.MaximumBid != m.MaximumBid {
		t.Errorf("Expected: %d, Got: %d", m.MaximumBid, m2.MaximumBid)
	}
	if m2.MaxTimestamp != m.MaxTimestamp {
		t.Errorf("Expected: %d, Got: %d", m.MaxTimestamp, m2.MaxTimestamp)
	}
}

func Test_EncodeDecodeOption(t *testing.T) {
	o := structs.Option{Expiration: "20150101", Strike: 10000, Symbol: "20150101AA100PUT", Time: int64(1000), Type: "p",
		Ask: 200, Bid: 100, IV: 1.111, Last: 150, OpenInterest: 1000, Underlying: "AA", Volume: 100}

	eo, err := Encode(&o, OptionEncodingOrder)

	if err != nil {
		t.Errorf("EncodeOption err: %s!", err)
	}

	o2 := structs.Option{}
	err = Decode(eo, &o2, OptionEncodingOrder)

	if err != nil {
		t.Errorf("DecodeOption err: %s!", err)
	}

	if o != o2 {
		t.Errorf("\n%v\n\nshould equal\n\n%v", o, o2)
	}
}

func Test_EncodeDecodeStock(t *testing.T) {
	s := structs.Stock{Symbol: "GOOG", Time: int64(1000), Ask: 200, Bid: 100, Last: 150, High: 300, Low: 50, Volume: 100}

	es, err := Encode(&s, StockEncodingOrder)

	if err != nil {
		t.Errorf("EncodeOption err: %s!", err)
	}

	s2 := structs.Stock{}
	err = Decode(es, &s2, StockEncodingOrder)

	if err != nil {
		t.Errorf("DecodeOption err: %s!", err)
	}

	if s != s2 {
		t.Errorf("\n%v\n\nshould equal\n\n%v", s, s2)
	}
}

func Test_OptionEncodingOrder(t *testing.T) {
	o := structs.Option{}
	r := reflect.ValueOf(&o).Elem()

	propmap := map[string]bool{}
	for _, propname := range OptionEncodingOrder {
		propmap[propname] = true
		if !r.FieldByName(propname).IsValid() {
			t.Errorf("Invalid Propname: %s", propname)
		}
	}
	if len(OptionEncodingOrder) < len(propmap) {
		t.Errorf("Expected %d, Got: %d", len(OptionEncodingOrder), len(propmap))
	}
	if len(propmap) < r.NumField() {
		t.Errorf("Expected %d, Got: %d", r.NumField(), len(propmap))
	}
}

func Test_getConfig(t *testing.T) {
	_, err := GetConfig("config")
	if err != nil {
		t.Errorf("Got err: %s!", err)
	}

	_, err = GetConfig("nonexistent")
	if err == nil {
		t.Errorf("Not expecting to find 'nonexistent' file.")
	}
}

func Test_SeekToNearestFriday(t *testing.T) {
	t1, _ := time.Parse("20060102", "20150107")
	t2 := SeekToNearestFriday(t1)

	if t2.Weekday() != time.Friday {
		t.Errorf("Expected Friday, Got: %v", t2.Weekday)
	}

	// Verify it returns same time if we are on a Friday.
	t1, _ = time.Parse("20060102", "20150109")
	t2 = SeekToNearestFriday(t1)

	if t2 != t1 {
		t.Errorf("Expected %v, Got %v", t1, t2)
	}
}

func Test_updateConfig(t *testing.T) {
	lines, _ := GetConfig("config")
	err := UpdateConfig("config", lines)
	if err != nil {
		t.Errorf("Got err: %s!", err)
	}
}

func Test_WeekID(t *testing.T) {
	ts := int64(1422022200)

	if ts == WeekID(ts) {
		t.Errorf("Expected ts != WeekID(ts)")
	}

	if WeekID(ts) != WeekID(WeekID(ts)) {
		t.Errorf("Expected  WeekID(ts) == WeekID(WeekID(ts))")
	}
}
