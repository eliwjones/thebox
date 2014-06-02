package tdameritrade

import (
	"io/ioutil"
	"strings"
	"testing"
)

func getConfig() (string, string, string, error) {
	b, err := ioutil.ReadFile("config")
	if err != nil {
		return "", "", "", err
	}
	lines := strings.Split(string(b), "\n")
	return lines[0], lines[1], lines[2], nil
}

func Test_getConfig(t *testing.T) {
	_, _, _, err := getConfig()
	if err != nil {
		t.Errorf("Got err: %s!", err)
	}
}

func Test_TDAmeritrade_Connect(t *testing.T) {
	id, pass, sid, err := getConfig()

	tda := &TDAmeritrade{Source: sid}

	token, err := tda.Connect(id, pass)
	if err != nil {
		t.Errorf("Got err: %s", err)
	}

	token, err = tda.Connect(id, "bad"+pass)
	if err == nil || token != "" {
		t.Errorf("Bad Pass should result in failure! Got err: %s, token: %s", err, token)
	}
}
