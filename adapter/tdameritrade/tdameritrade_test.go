package tdameritrade

import (
	"github.com/eliwjones/thebox/util/funcs"
	"github.com/eliwjones/thebox/util/interfaces"

	"testing"
)

var (
	jsessionid = ""
	gtda       *TDAmeritrade
)

func Test_TDAmeritrade_Connect(t *testing.T) {
	id, pass, sid, jsess, _ := funcs.GetConfig()

	tda := &TDAmeritrade{Source: sid}

	token, err := tda.Connect(id, pass, jsess)
	if err != nil {
		t.Errorf("Got err: %s", err)
	} else {
		jsessionid = token
		if jsessionid != jsess {
			// Update Config.
			funcs.UpdateConfig(id, pass, sid, jsessionid)
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
	id, pass, sid, _, _ := funcs.GetConfig()
	tda := New(id, pass, sid, jsessionid)

	gtda = tda
}

func Test_TDAmeritrade_Adapter(t *testing.T) {
	id, pass, sid, _, _ := funcs.GetConfig()

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

func Test_TDAmeritrade_GetOptions(t *testing.T) {
	underlying := "INTC"
	options, err := gtda.GetOptions(underlying)
	if err != nil {
		t.Errorf("Got err: %s", err)
	}
	for _, option := range options {
		if option.Type != "p" && option.Type != "c" {
			t.Errorf("Expected type 'p' or 'c'. Got: %s", option.Type)
		}
		if option.Strike == 0 {
			t.Errorf("0 is not a valid strike.")
		}
		if option.Expiration == "" {
			t.Errorf("Expiration is empty.")
		}
		if option.Symbol == "" {
			t.Errorf("Symbol does not look right.")
		}
		if option.Underlying != underlying {
			t.Errorf("Expected: %s. Got: %s", underlying, option.Underlying)
		}
	}
}

func Test_TDAmeritrade_GetPositions(t *testing.T) {
	_, err := gtda.GetPositions()
	if err != nil {
		t.Errorf("Got err: %s", err)
	}
}
