package velocorev2stable

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/common"
)

type Metadata struct {
	Offset int `json:"offset"`
}

type bytes32 [32]byte

type Extra struct {
	Amp             *big.Int             `json:"amp"`
	Fee1e18         *big.Int             `json:"fee1e18"`
	LpTokenBalances map[string]*big.Int  `json:"lpTokenBalances"`
	TokenInfo       map[string]tokenInfo `json:"tokenInfo"`
}

type tokenInfo struct {
	Scale uint8 `json:"scale"`
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
