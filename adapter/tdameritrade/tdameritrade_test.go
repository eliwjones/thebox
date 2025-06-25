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
	/*
		// TDAmeritrade API has gone away.

		lines, _ := funcs.GetConfig("test_config")
		id := lines[0]
		pass := lines[1]
		sid := lines[2]
		jsess := lines[3]

		tda := &TDAmeritrade{Source: sid}

		token, err := tda.Connect(id, pass, jsess)
		if err != nil {
			t.Errorf("Got err: %s", err)
		} else {
			jsessionid = token
			if jsessionid != jsess {
				// Update Config.
				funcs.UpdateConfig("test_config", []string{id, pass, sid, jsessionid})
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
		token, _ = tda.Connect(id, pass, badjsessionid)
		if token == badjsessionid {
			t.Errorf("Expected valid token instead of badjsessionid: %s", badjsessionid)
		}
	*/
}

func Test_TDAmeritrade_New(t *testing.T) {
	lines, _ := funcs.GetConfig("test_config")
	id := lines[0]
	pass := lines[1]
	sid := lines[2]

	tda := New(id, pass, sid, jsessionid)

	gtda = tda
}

func Test_TDAmeritrade_Adapter(t *testing.T) {
	lines, _ := funcs.GetConfig("test_config")
	id := lines[0]
	pass := lines[1]
	sid := lines[2]

	var a interfaces.Adapter
	a = New(id, pass, sid, jsessionid)
	if a == nil {
		t.Errorf("%+v", a)
	}
}

func Test_TDAmeritrade_GetBalances(t *testing.T) {
	/*
		  // TDAmeritrade API has gone away.
		_, err := gtda.GetBalances()
		if err != nil {
			t.Errorf("Got err: %s", err)
		}
	*/
}

func Test_TDAmeritrade_GetOptions(t *testing.T) {
	/*
			  // TDAmeritrade API has gone away.
		underlying := "GOOG"
		options, stock, err := gtda.GetOptions(underlying, funcs.NextFriday(time.Now()).Format("200601"))
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

		s := structs.Stock{}
		if s == stock {
			t.Errorf("Got back zero stock.")
		}

		if stock.Volume == 0 {
			t.Errorf("Stock volume is 0.")
		}
	*/
}

func Test_TDAmeritrade_GetPositions(t *testing.T) {
	/*
			  // TDAmeritrade API has gone away.
		_, err := gtda.GetPositions()
		if err != nil {
			t.Errorf("Got err: %s", err)
		}
	*/
}
