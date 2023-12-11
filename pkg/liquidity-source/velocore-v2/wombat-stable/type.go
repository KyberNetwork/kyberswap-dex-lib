package wombatstable

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type Metadata struct {
	Offset int `json:"offset"`
}

type Meta struct {
	Vault    string            `json:"vault"`
	Wrappers map[string]string `json:"wrappers"`
}

type Gas struct {
	SwapNoConvert  int64
	SwapConvertIn  int64
	SwapConvertOut int64
}

type bytes32 [32]byte

type Extra struct {
	Amp             *big.Int             `json:"amp"`
	Fee1e18         *big.Int             `json:"fee1e18"`
	LpTokenBalances map[string]*big.Int  `json:"lpTokenBalances"`
	TokenInfo       map[string]tokenInfo `json:"tokenInfo"`
}

type StaticExtra struct {
	Vault    string            `json:"vault"`
	Wrappers map[string]string `json:"wrappers"`
}

type tokenInfo struct {
	IndexPlus1 uint8          `json:"indexPlus1"`
	Scale      uint8          `json:"scale"`
	Gauge      common.Address `json:"-"`
}

type poolData struct {
	Tokens          []*entity.PoolToken
	PoolReserves    entity.PoolReserves
	Amp             *big.Int
	Fee1e18         *big.Int
	LpTokenBalances map[string]*big.Int
}

// rpc

type poolDataResp struct {
	Data struct {
		Pool           common.Address
		PoolType       string
		LpTokens       []bytes32
		MintedLPTokens []*big.Int
		ListedTokens   []bytes32
		Reserves       []*big.Int
		PoolParams     []byte
	}
}
