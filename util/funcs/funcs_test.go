package funcs

import (
	"testing"
)

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
