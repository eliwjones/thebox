package interfaces

import (
	"github.com/eliwjones/thebox/util"
	"github.com/eliwjones/thebox/util/structs"
)

type Adapter interface {
	ClosePosition(id string, limit int) error                                         // Close out an open position.
	Commission() map[util.ContractType]map[string]int                                 // Commission information.
	ContractMultiplier() map[util.ContractType]int                                    // How many contracts trade per type.  Generally 1 for Stocks and 100 for Options.
	Connect(id string, auth string, token string) (string, error)                     // Connect.
	GetBalances() (map[string]int, error)                                             // Returns values in cents, "cash", "value" ("stock"? "option"?)
	GetOptions(symbol string, expire string) ([]structs.Option, structs.Stock, error) // Get Options for a given symbol and expiration month.
	GetOrders(filter string) (map[string]structs.Order, error)                        // "open", "filled"
	GetPositions() (map[string]structs.Position, error)                               // Return curren view of Positions.
	Reset()                                                                           // Reset all orders, positions, and value.
	SubmitOrder(order structs.Order) (string, error)
}
