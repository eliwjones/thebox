package funcs

import (
	"github.com/eliwjones/thebox/util/structs"

	"reflect"
	"testing"
)

func Test_EncodeDecodeOption(t *testing.T) {
	o := structs.Option{Expiration: "20150101", Strike: 10000, Symbol: "20150101AA100PUT", Time: "12:00", Type: "p",
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

func Test_updateConfig(t *testing.T) {
	lines, _ := GetConfig("config")
	err := UpdateConfig("config", lines)
	if err != nil {
		t.Errorf("Got err: %s!", err)
	}
}
