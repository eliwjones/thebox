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
)

const (
	BASEURL = "https://apis.tdameritrade.com"
)

type XmlLogIn struct {
	LoginResult LoginResult `xml:"xml-log-in"`
}

type LoginResult struct {
	SessionId string `xml:"session-id"`
}

type TDAmeritrade struct {
	Id         string // username
	Auth       string // password or whatnot.
	Source     string // App Source ID.
	JsessionID string // Return from Login inside ['amtd']['xml-log-in']['session-id']

	Tables map[string]int // "position", "order", "cash", "value" ... "margin"?

	// Lcal cache.
	Positions map[string]structs.Position // most likely just util.Positions.
	Orders    map[string]structs.Order    // most likely just util.Orders.
	Cash      int                         // cash available.
	Value     int                         // total account value (cash + position value).
}

func New(id string, auth string, source string) *TDAmeritrade {
	s := &TDAmeritrade{Id: id, Auth: auth, Source: source}

	s.JsessionID, _ = s.Connect(s.Id, s.Auth)
	s.Tables = map[string]int{"position": 1, "order": 1, "cash": 1, "value": 1}

	// Mocked data.  Not about to make actual http api to simulate external resource.
	s.Cash = 1000000 * 100 // $1 million in cents.
	s.Value = s.Cash
	s.Positions = map[string]structs.Position{}
	s.Orders = map[string]structs.Order{}

	return s
}

func (s *TDAmeritrade) Connect(id string, auth string) (string, error) {
	data := url.Values{}
	data.Set("userid", id)
	data.Set("password", auth)
	data.Set("source", s.Source)
	data.Set("version", "1.0")

	r, err := http.NewRequest("POST", BASEURL+"/apps/100/LogIn", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", errors.New("http.NewRequest() Failed for user: %s, auth: %s!")
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := XmlLogIn{}
	err = xml.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	sessionID := result.LoginResult.SessionId
	if sessionID == "" {
		return "", errors.New(string(body))
	}
	return sessionID, nil
}
