package prop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type AssetReserves struct {
	Tokens   []common.Address
	Balances []*big.Int
}

type Extra struct {
	Samples [][][2]*big.Int `json:"samples"`
	MaxIn   []*big.Int      `json:"maxIn,omitempty"`
}

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

type PoolMetaInfo struct {
	BlockNumber   uint64 `json:"blockNumber"`
	RouterAddress string `json:"routerAddress"`
}
