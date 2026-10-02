package prop

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type AssetReserves struct {
	Tokens   []common.Address
	Balances []uint256.Int
}

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}
