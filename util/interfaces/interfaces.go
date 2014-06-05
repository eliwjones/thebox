package interfaces

import (
	"github.com/eliwjones/thebox/util/structs"
)

type Adapter interface {
	Connect(id string, auth string, token string) (string, error)
	GetBalances() (map[string]int, error)                      // Returns values in cents, "cash", "value" ("stock"? "option"?)
	GetOrders(filter string) (map[string]structs.Order, error) // "open", "filled"
	GetPositions() (map[string]structs.Position, error)
	SubmitOrder(order structs.Order) (string, error)
}
