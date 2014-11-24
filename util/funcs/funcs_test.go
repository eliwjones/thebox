package funcs

import (
	"github.com/eliwjones/thebox/util/structs"

	"reflect"
	"testing"
)

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
