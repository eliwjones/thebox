package simulate

import (
	"github.com/eliwjones/thebox/util"

	"errors"
	"fmt"
	"math/rand"
)

const (
	TOKEN = "thisisanaccesstoken"
)

type Simulate struct {
	Id     string         // username
	Auth   string         // password or whatnot.
	Token  string         // account access token. (most likely oauth.)
	Tables map[string]int // "position", "order", "cash", "value" ... "margin"?

	// Mocks.
	Positions map[string]util.Position // most likely just util.Positions.
	Orders    map[string]util.Order    // most likely just util.Orders.
	Cash      int                      // cash available.
	Value     int                      // total account value (cash + position value).
}

func New(id string, auth string) *Simulate {
	s := &Simulate{Id: id, Auth: auth}
	s.Token, _ = s.Connect(s.Id, s.Auth)
	s.Tables = map[string]int{"position": 1, "order": 1, "cash": 1, "value": 1}

	// Mocked data.  Not about to make actual http api to simulate external resource.
	s.Cash = 1000000 * 100 // $1 million in cents.
	s.Value = s.Cash
	s.Positions = map[string]util.Position{}
	s.Orders = map[string]util.Order{}

	return s
}

func (s *Simulate) Connect(id string, auth string) (string, error) {
	if id != "simulate" || auth != "simulation" {
		return "", errors.New("Auth Failed for user: %s, auth: %s!")
	}
	return TOKEN, nil
}

func (s *Simulate) Get(table string, key string) (interface{}, error) {
	if s.Token != TOKEN {
		return nil, errors.New("Bad Auth Token!")
	}
	if s.Tables[table] != 1 {
		return nil, fmt.Errorf("Invalid table: %s! Choose from: %+v", table, s.Tables)
	}
	switch table {
	case "position":
		p, exists := s.Positions[key]
		if !exists {
			return nil, fmt.Errorf("No Position found for key: %s!", key)
		}
		return p, nil
	case "order":
		o, exists := s.Orders[key]
		if !exists {
			return nil, fmt.Errorf("No Order found for key: %s!", key)
		}
		return o, nil
	case "cash":
		return s.Cash, nil
	case "value":
		return s.Value, nil
	}

	return nil, fmt.Errorf("No data found for key: %s!", key)
}

func (s *Simulate) GetOrders(filter string) (map[string]util.Order, error) {
	if s.Token != TOKEN {
		return nil, errors.New("Bad Auth Token!")
	}
	// More complex api call and munging goes here.
	return s.Orders, nil
}

func (s *Simulate) GetPositions() (map[string]util.Position, error) {
	if s.Token != TOKEN {
		return nil, errors.New("Bad Auth Token!")
	}
	// More complex api call and munging goes here.
	return s.Positions, nil
}

func (s *Simulate) SubmitOrder(order util.Order) (string, error) {
	if s.Token != TOKEN {
		return "", errors.New("Bad Auth Token!")
	}
	orderid := fmt.Sprintf("order-%d", rand.Intn(1000000))
	s.Orders[orderid] = order
	return orderid, nil
}
