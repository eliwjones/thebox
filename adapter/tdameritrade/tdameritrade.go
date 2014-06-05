package tdameritrade

import (
	"github.com/eliwjones/thebox/util/structs"

	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	BASEURL = "https://apis.tdameritrade.com"
)

// Lazy Kitchen Sink Struct.
type TDAResponse struct {
	SessionId string `xml:"xml-log-in>session-id"`
	AccountId string `xml:"xml-log-in>associated-account-id"`

	AvailableFunds string `xml:"balance>available-funds-for-trading"`
	AccountValue   string `xml:"balance>account-value>current"`

	StockPositions  []Position `xml:"positions>stocks>position"`
	OptionPositions []Position `xml:"positions>options>position"`
}

type Position struct {
	Symbol string `xml:"security>symbol"`
	Volume string `xml:"quantity"`
	Value  string `xml:"current-value"`
}

type TDAmeritrade struct {
	Id         string // username
	Auth       string // password or whatnot.
	Source     string // App Source ID.
	JsessionID string // Return from Login inside ['amtd']['xml-log-in']['session-id']

	Tables map[string]int // "position", "order", "cash", "value" ... "margin"?

	// Local cache.
	Positions map[string]structs.Position // most likely just util.Positions.
	Orders    map[string]structs.Order    // most likely just util.Orders.
	Cash      int                         // cash available.
	Value     int                         // total account value (cash + position value).
}

func New(id string, auth string, source string, jsessionid string) *TDAmeritrade {
	s := &TDAmeritrade{Id: id, Auth: auth, Source: source}

	s.JsessionID, _ = s.Connect(s.Id, s.Auth, jsessionid)

	s.Tables = map[string]int{"position": 1, "order": 1, "cash": 1, "value": 1}

	resources, _ := s.GetBalances()
	s.Cash = resources["cash"]
	s.Value = resources["value"]
	s.Positions = map[string]structs.Position{}
	s.Orders = map[string]structs.Order{}

	return s
}

func (s *TDAmeritrade) Connect(id string, auth string, jsessionid string) (string, error) {
	if jsessionid != "" {
		params := map[string]string{"source": s.Source}
		body, _ := request(BASEURL+"/apps/KeepAlive"+";jsessionid="+jsessionid, "GET", params)
		if string(body) == "LoggedOn" {
			return jsessionid, nil
		}

	}
	params := map[string]string{"userid": id, "password": auth, "source": s.Source, "version": "1.0"}
	body, err := request(BASEURL+"/apps/100/LogIn", "POST", params)
	if err != nil {
		return "", err
	}
	result := TDAResponse{}
	err = xml.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	sessionID := result.SessionId
	if sessionID == "" {
		return "", errors.New(string(body))
	}
	return sessionID, nil
}

func (s *TDAmeritrade) GetBalances() (map[string]int, error) {
	params := map[string]string{"source": s.Source, "type": "b"}
	body, err := request(BASEURL+"/apps/100/BalancesAndPositions"+";jsessionid="+s.JsessionID, "GET", params)
	if err != nil {
		return map[string]int{"cash": s.Cash, "value": s.Value}, err
	}
	result := TDAResponse{}
	err = xml.Unmarshal(body, &result)
	if err != nil {
		return map[string]int{"cash": s.Cash, "value": s.Value}, err
	}
	cash, err := strconv.ParseFloat(result.AvailableFunds, 64)
	if err != nil {
		return map[string]int{"cash": s.Cash, "value": s.Value}, err
	}
	value, err := strconv.ParseFloat(result.AccountValue, 64)
	if err != nil {
		return map[string]int{"cash": s.Cash, "value": s.Value}, err
	}
	// Convert to cents and return int.
	return map[string]int{"cash": int(cash * 100), "value": int(value * 100)}, nil
}

func (s *TDAmeritrade) GetOrders(filter string) (map[string]structs.Order, error) {
	return s.Orders, nil
}

func (s *TDAmeritrade) GetPositions() (map[string]structs.Position, error) {
	params := map[string]string{"source": s.Source, "type": "p"}
	body, err := request(BASEURL+"/apps/100/BalancesAndPositions"+";jsessionid="+s.JsessionID, "GET", params)
	if err != nil {
		return s.Positions, err
	}
	result := TDAResponse{}
	err = xml.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return s.Positions, nil
}

func (s *TDAmeritrade) SubmitOrder(order structs.Order) (string, error) {
	return "", nil
}

func request(urlStr string, method string, params map[string]string) ([]byte, error) {
	b := bytes.NewBufferString("")

	data := url.Values{}
	for id, value := range params {
		data.Set(id, value)
	}
	encodedParams := strings.Replace(data.Encode(), "~", "%7E", -1) // Go does not Encode() tilde.

	switch method {
	case "POST":
		b = bytes.NewBufferString(encodedParams)
	case "GET":
		if encodedParams != "" {
			urlStr += "?" + encodedParams
		}
	}
	r, err := http.NewRequest(method, urlStr, b)
	if err != nil {
		return nil, err
	}
	switch method {
	case "POST":
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Add("Content-Length", strconv.Itoa(len(encodedParams)))
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
