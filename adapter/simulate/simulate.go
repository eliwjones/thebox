package simulate

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
	"fmt"
	"math/rand"
)

const (
	TOKEN = "thisisanaccesstoken"
)

type Simulate struct {
	Id                 string                               // username
	Auth               string                               // password or whatnot.
	commission         map[util.ContractType]map[string]int // Commission information.
	contractMultiplier map[util.ContractType]int            // How many contracts trade per unit of volume.  Generally 1 for stocks and 100 for options.
	Token              string                               // account access token. (most likely oauth.)
	Tables             map[string]int                       // "position", "order", "cash", "value" ... "margin"?

	// Mocks.
	Positions map[string]structs.Position // most likely just util.Positions.
	Orders    map[string]structs.Order    // most likely just util.Orders.
	Cash      int                         // cash available.
	Value     int                         // total account value (cash + position value).
}

func New(id string, auth string) *Simulate {
	s := &Simulate{Id: id, Auth: auth}

	s.contractMultiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	s.commission = map[util.ContractType]map[string]int{}
	s.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	s.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	s.Token, _ = s.Connect(s.Id, s.Auth, "")
	s.Tables = map[string]int{"position": 1, "order": 1, "cash": 1, "value": 1}

	// Mocked data.  Not about to make actual http api to simulate external resource.
	s.Cash = 300000 * 100 // $300k in cents.
	s.Value = s.Cash
	s.Positions = map[string]structs.Position{}
	s.Orders = map[string]structs.Order{}

	// Process pulses.. sift through open orders determine fills?
	// Or, just, auto-fill orders and call it a day?
	// Only thing adapter needs to worry about now is +- cash for order value and commissions.
	//   1. Move order into position if it isn't already there.
	//   2.
	// No need for Value updating just yet.

	return s
}

func (s *Simulate) ClosePosition(id string, limit int) error {
	p, exists := s.Positions[id]
	if !exists {
		return fmt.Errorf("PositionID: %s, Not found!", id)
	}
	// take commission.. transfer value to cash.
	commission := s.orderCommission(p.Order)
	s.Cash -= commission
	s.Value -= commission

	// Current value of the position gets released to Cash.
	value := p.Order.Volume * s.contractMultiplier[p.Order.Type] * limit
	s.Cash += value

	// Delta gets merged into Value.
	startValue := p.Order.Volume * s.contractMultiplier[p.Order.Type] * p.Order.Limitprice
	delta := value - startValue
	s.Value += delta

	delete(s.Positions, p.Id)

	return nil
}

func (s *Simulate) Commission() map[util.ContractType]map[string]int {
	return s.commission
}

func (s *Simulate) Connect(id string, auth string, token string) (string, error) {
	if id != "simulate" || auth != "simulation" {
		return "", errors.New("Auth Failed for user: %s, auth: %s!")
	}
	return TOKEN, nil
}

func (s *Simulate) ContractMultiplier() map[util.ContractType]int {
	return s.contractMultiplier
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

func (s *Simulate) GetBalances() (map[string]int, error) {
	if s.Token != TOKEN {
		return nil, errors.New("Bad Auth Token!")
	}
	// More complex api call and munging goes here.
	return map[string]int{"cash": s.Cash, "value": s.Value}, nil
}

func (s *Simulate) GetOrders(filter string) (map[string]structs.Order, error) {
	if s.Token != TOKEN {
		return nil, errors.New("Bad Auth Token!")
	}
	// More complex api call and munging goes here.
	return s.Orders, nil
}

func (s *Simulate) GetPositions() (map[string]structs.Position, error) {
	if s.Token != TOKEN {
		return nil, errors.New("Bad Auth Token!")
	}
	// More complex api call and munging goes here.
	return s.Positions, nil
}

func (s *Simulate) orderCommission(o structs.Order) int {
	return s.commission[o.Type]["base"] + o.Volume*s.commission[o.Type]["unit"]
}

func (s *Simulate) SubmitOrder(order structs.Order) (string, error) {
	if s.Token != TOKEN {
		return "", errors.New("Bad Auth Token!")
	}
	orderid := fmt.Sprintf("order-%d", rand.Intn(1000000))
	order.Id = orderid
	s.Orders[orderid] = order

	// Fill immediately.
	// Transfer from Cash to Value.
	value := order.Volume * s.contractMultiplier[order.Type] * order.Limitprice
	s.Cash -= value

	// Commission disappears in a puff of smoke.
	commission := s.orderCommission(order)
	s.Cash -= commission
	s.Value -= commission

	// Delete order, Add position.
	delete(s.Orders, orderid)
	p := structs.Position{Id: order.Id, Order: order, Fillprice: order.Limitprice}
	s.Positions[p.Id] = p

	return orderid, nil
}
