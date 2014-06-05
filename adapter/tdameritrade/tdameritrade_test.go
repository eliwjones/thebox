package tdameritrade

import (
	"github.com/eliwjones/thebox/util/interfaces"

	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

var (
	jsessionid = ""
	gtda       *TDAmeritrade
)

func getConfig() (string, string, string, string, error) {
	b, err := ioutil.ReadFile("config")
	if err != nil {
		return "", "", "", "", err
	}
	lines := strings.Split(string(b), "\n")
	return lines[0], lines[1], lines[2], lines[3], nil
}

func updateConfig(id string, pass string, sid string, jsess string) error {
	f := []byte(fmt.Sprintf("%s\n%s\n%s\n%s", id, pass, sid, jsess))
	err := ioutil.WriteFile("config", f, 0777)
	return err
}

func Test_getConfig(t *testing.T) {
	_, _, _, _, err := getConfig()
	if err != nil {
		t.Errorf("Got err: %s!", err)
	}
}

func Test_updateConfig(t *testing.T) {
	id, pass, sid, jsess, _ := getConfig()
	err := updateConfig(id, pass, sid, jsess)
	if err != nil {
		t.Errorf("Got err: %s!", err)
	}
}

func Test_TDAmeritrade_Connect(t *testing.T) {
	id, pass, sid, jsess, _ := getConfig()

	tda := &TDAmeritrade{Source: sid}

	token, err := tda.Connect(id, pass, jsess)
	if err != nil {
		t.Errorf("Got err: %s", err)
	} else {
		jsessionid = token
		if jsessionid != jsess {
			// Update Config.
			updateConfig(id, pass, sid, jsessionid)
		}
	}

	token, err = tda.Connect(id, "bad"+pass, "")
	if err == nil || token != "" {
		t.Errorf("Bad Pass should result in failure! Got err: %s, token: %s", err, token)
	}

	token, _ = tda.Connect(id, pass, jsessionid)
	if token != jsessionid {
		t.Errorf("Expected token: %s to equal jsessionid: %s", token, jsessionid)
	}
	badjsessionid := "thisaintright"
	token, err = tda.Connect(id, pass, badjsessionid)
	if token == badjsessionid {
		t.Errorf("Expected valid token instead of badjsessionid: %s", badjsessionid)
	}
}

func Test_TDAmeritrade_New(t *testing.T) {
	id, pass, sid, _, _ := getConfig()
	tda := New(id, pass, sid, jsessionid)

	gtda = tda
}

func Test_TDAmeritrade_Adapter(t *testing.T) {
	id, pass, sid, _, _ := getConfig()

	var a interfaces.Adapter
	a = New(id, pass, sid, jsessionid)
	if a == nil {
		t.Errorf("%+v", a)
	}
}

func Test_TDAmeritrade_GetBalances(t *testing.T) {
	_, err := gtda.GetBalances()
	if err != nil {
		t.Errorf("Got err: %s", err)
	}
}

func Test_TDAmeritrade_GetPositions(t *testing.T) {
	_, err := gtda.GetPositions()
	if err != nil {
		t.Errorf("Got err: %s", err)
	}
}
