package gyroeclp

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Gas struct {
	Swap int64
}

type StaticExtra struct {
	PoolID        string `json:"poolId"`
	PoolType      string `json:"poolType"`
	PoolTypeVer   int    `json:"poolTypeVersion"`
	TokenDecimals []int  `json:"tokenDecimals"`
	Vault         string `json:"vault"`
}

type Extra struct {
	Paused            bool           `json:"paused"`
	ScalingFactors    []*uint256.Int `json:"scalingFactors"`
	SwapFeePercentage *uint256.Int   `json:"swapFeePercentage"`
	ParamsAlpha       *int256.Int    `json:"paramsAlpha"`
	ParamsBeta        *int256.Int    `json:"paramsBeta"`
	ParamsC           *int256.Int    `json:"paramsC"`
	ParamsS           *int256.Int    `json:"paramsS"`
	ParamsLambda      *int256.Int    `json:"paramsLambda"`
	TauAlphaX         *int256.Int    `json:"tauAlphaX"`
	TauAlphaY         *int256.Int    `json:"tauAlphaY"`
	TauBetaX          *int256.Int    `json:"tauBetaX"`
	TauBetaY          *int256.Int    `json:"tauBetaY"`
	U                 *int256.Int    `json:"u"`
	V                 *int256.Int    `json:"v"`
	W                 *int256.Int    `json:"w"`
	Z                 *int256.Int    `json:"z"`
	DSq               *int256.Int    `json:"dSq"`
}

type TokenRatesResp struct {
	Rate0 *big.Int
	Rate1 *big.Int
}

type PoolTokensResp struct {
	Tokens          []common.Address
	Balances        []*big.Int
	LastChangeBlock *big.Int
}

type PausedStateResp struct {
	Paused              bool
	PauseWindowEndTime  *big.Int
	BufferPeriodEndTime *big.Int
}

type ECLPParamsResp struct {
	Params struct {
		Alpha  *big.Int
		Beta   *big.Int
		C      *big.Int
		S      *big.Int
		Lambda *big.Int
	}

	D struct {
		TauAlpha struct {
			X *big.Int
			Y *big.Int
		}
		TauBeta struct {
			X *big.Int
			Y *big.Int
		}
		U   *big.Int
		V   *big.Int
		W   *big.Int
		Z   *big.Int
		DSq *big.Int
	}
}

type rpcResp struct {
	PoolTokens        PoolTokensResp
	SwapFeePercentage *big.Int
	PausedState       PausedStateResp
	TokenRatesResp    TokenRatesResp
	ECLPParamsResp    ECLPParamsResp
	BlockNumber       uint64
}
