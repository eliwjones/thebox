package tdameritrade

import (
	"github.com/eliwjones/thebox/util/structs"

	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	BASEURL = "https://apis.tdameritrade.com"
)

type LoginResult struct {
	SessionId string `xml:"xml-log-in>session-id"`
	AccountId string `xml:"xml-log-in>associated-account-id"`
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

func New(id string, auth string, source string) *TDAmeritrade {
	s := &TDAmeritrade{Id: id, Auth: auth, Source: source}

	//s.JsessionID, _ = s.Connect(s.Id, s.Auth)
	s.Tables = map[string]int{"position": 1, "order": 1, "cash": 1, "value": 1}

	// Mocked data.  Not about to make actual http api to simulate external resource.
	s.Cash = 1000000 * 100 // $1 million in cents.
	s.Value = s.Cash
	s.Positions = map[string]structs.Position{}
	s.Orders = map[string]structs.Order{}

	return s
}

func (s *TDAmeritrade) Connect(id string, auth string) (string, error) {
	params := map[string]string{"userid": id, "password": auth, "source": s.Source, "version": "1.0"}
	body, err := request(BASEURL+"/apps/100/LogIn", "POST", params)
	if err != nil {
		return "", err
	}
	result := LoginResult{}
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

func (s *TDAmeritrade) GetPositions() (map[string]structs.Position, error) {
	// Will have to just go through /apps/100/BalancesAndPositions;jsessionid=BLAH?source=BLAH
	params := map[string]string{"source": s.Source}
	_, err := request(BASEURL+"/apps/100/BalancesAndPositions"+";jsessionid="+s.JsessionID, "GET", params)
	if err != nil {
		return s.Positions, err
	}
	return s.Positions, nil
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
	fmt.Printf("BODY:\n%s\n", string(body))
	return body, nil
}
