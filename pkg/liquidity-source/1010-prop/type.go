package prop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type AssetReserves struct {
	Tokens   []common.Address
	Balances []*big.Int
}

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}
