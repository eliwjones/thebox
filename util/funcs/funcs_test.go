package funcs

import (
	"github.com/eliwjones/thebox/util/structs"

	"reflect"
	"testing"
)

func Test_EncodeDecodeOption(t *testing.T) {
	o := structs.Option{Expiration: "20150101", Strike: 10000, Symbol: "20150101AA100PUT", Time: "12:00", Type: "p",
		Ask: 200, Bid: 100, IV: 1.111, Last: 150, OpenInterest: 1000, Underlying: "AA", Volume: 100}

	eo, err := EncodeOption(o)

	if err != nil {
		t.Errorf("EncodeOption err: %s!", err)
	}

	o2, err := DecodeOption(eo)

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
	_, _, _, _, err := GetConfig()
	if err != nil {
		t.Errorf("Got err: %s!", err)
	}
}

func Test_updateConfig(t *testing.T) {
	id, pass, sid, jsess, _ := GetConfig()
	err := UpdateConfig(id, pass, sid, jsess)
	if err != nil {
		t.Errorf("Got err: %s!", err)
	}
}
