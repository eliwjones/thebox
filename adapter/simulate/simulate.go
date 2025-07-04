package simulate

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"

	"errors"
	"fmt"
	"math/rand"
	"time"
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

func New(id string, auth string, cash int) *Simulate {
	s := &Simulate{Id: id, Auth: auth}

	s.contractMultiplier = map[util.ContractType]int{util.OPTION: 100, util.STOCK: 1}
	s.commission = map[util.ContractType]map[string]int{}
	s.commission[util.OPTION] = map[string]int{"base": 999, "unit": 75}
	s.commission[util.STOCK] = map[string]int{"base": 999, "unit": 0}

	s.Token, _ = s.Connect(s.Id, s.Auth, "")
	s.Tables = map[string]int{"position": 1, "order": 1, "cash": 1, "value": 1}

	// Mocked data.  Not about to make actual http api to simulate external resource.
	s.Cash = cash
	s.Reset()

	// Process pulses.. sift through open orders determine fills?
	// Or, just, auto-fill orders and call it a day?
	// Only thing adapter needs to worry about now is +- cash for order value and commissions.
	//   1. Move order into position if it isn't already there.
	//   2.
	// No need for Value updating just yet.

	return s
}

func (s *Simulate) Reset() {
	// Called when crossing week boundaries.
	s.Value = s.Cash
	s.Positions = map[string]structs.Position{}
	s.Orders = map[string]structs.Order{}
}

func (s *Simulate) ClosePosition(id string, limit int) error {
	p, exists := s.Positions[id]
	if !exists {
		return fmt.Errorf("positionID: %s, not found", id)
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
		return "", fmt.Errorf("auth failed for user: %s, auth: %s", id, auth)
	}
	return TOKEN, nil
}

func (s *Simulate) ContractMultiplier() map[util.ContractType]int {
	return s.contractMultiplier
}

func (s *Simulate) Get(table string, key string) (any, error) {
	if s.Token != TOKEN {
		return nil, errors.New("bad auth token")
	}
	if s.Tables[table] != 1 {
		return nil, fmt.Errorf("invalid table: %s choose from: %+v", table, s.Tables)
	}
	switch table {
	case "position":
		p, exists := s.Positions[key]
		if !exists {
			return nil, fmt.Errorf("no position found for key: %s", key)
		}
		return p, nil
	case "order":
		o, exists := s.Orders[key]
		if !exists {
			return nil, fmt.Errorf("no order found for key: %s", key)
		}
		return o, nil
	case "cash":
		return s.Cash, nil
	case "value":
		return s.Value, nil
	}

	return nil, fmt.Errorf("no data found for key: %s", key)
}

func (s *Simulate) GetBalances() (map[string]int, error) {
	if s.Token != TOKEN {
		return nil, errors.New("bad auth token")
	}
	// More complex api call and munging goes here.
	return map[string]int{"cash": s.Cash, "value": s.Value}, nil
}

func (s *Simulate) GetOptions(symbol string, month string) ([]structs.Option, structs.Stock, error) {
	stock := structs.Stock{
		Symbol: symbol,
		Bid:    15000,
		Ask:    15010,
		Time:   time.Now().UTC().Unix(),
	}

	expirationDate := month + "15"

	options := []structs.Option{
		{
			Symbol:     fmt.Sprintf("%s_%sC155", symbol, month),
			Underlying: symbol,
			Strike:     15500,
			Expiration: expirationDate,
			Type:       "c",
			Bid:        50,
			Ask:        55,
		},
		{
			Symbol:     fmt.Sprintf("%s_%sP145", symbol, month),
			Underlying: symbol,
			Strike:     14500,
			Expiration: expirationDate,
			Type:       "p",
			Bid:        40,
			Ask:        45,
		},
	}

	return options, stock, nil
}

func (s *Simulate) GetOrders(filter string) (map[string]structs.Order, error) {
	if s.Token != TOKEN {
		return nil, errors.New("bad auth token")
	}
	// More complex api call and munging goes here.
	return s.Orders, nil
}

func (s *Simulate) GetPositions() (map[string]structs.Position, error) {
	if s.Token != TOKEN {
		return nil, errors.New("bad auth token")
	}
	// More complex api call and munging goes here.
	return s.Positions, nil
}

func (s *Simulate) orderCommission(o structs.Order) int {
	return s.commission[o.Type]["base"] + o.Volume*s.commission[o.Type]["unit"]
}

func (s *Simulate) SubmitOrder(order structs.Order) (string, error) {
	if s.Token != TOKEN {
		return "", errors.New("bad auth token")
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
	p := structs.Position{Id: order.Id, Order: order, Fillprice: order.Limitprice, Commission: commission}
	s.Positions[p.Id] = p

	return orderid, nil
}
