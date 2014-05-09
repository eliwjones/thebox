package interfaces

import (
	"github.com/eliwjones/thebox/util/structs"
)

type Adapter interface {
	Connect(id string, auth string) (string, error)
	Get(table string, key string) (interface{}, error)
	GetOrders(filter string) (map[string]structs.Order, error) // "open", "filled"
	GetPositions() (map[string]structs.Position, error)
	SubmitOrder(order structs.Order) (string, error)
}
