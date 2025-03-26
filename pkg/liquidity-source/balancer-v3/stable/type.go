package stable

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type Extra struct {
	HooksConfig                shared.HooksConfig    `json:"hooksConfig"`
	StaticSwapFeePercentage    *uint256.Int          `json:"staticSwapFeePercentage"`
	AggregateSwapFeePercentage *uint256.Int          `json:"aggregateSwapFeePercentage"`
	AmplificationParameter     *uint256.Int          `json:"amplificationParameter"`
	BalancesLiveScaled18       []*uint256.Int        `json:"balancesLiveScaled18"`
	DecimalScalingFactors      []*uint256.Int        `json:"decimalScalingFactors"`
	TokenRates                 []*uint256.Int        `json:"tokenRates"`
	Buffers                    []*shared.ExtraBuffer `json:"buffers"`
	IsVaultPaused              bool                  `json:"isVaultPaused,omitempty"`
	IsPoolPaused               bool                  `json:"isPoolPaused,omitempty"`
	IsPoolInRecoveryMode       bool                  `json:"isPoolInRecoveryMode,omitempty"`
}

type AmplificationParameter struct {
	Value      *big.Int
	IsUpdating bool
	Precision  *big.Int
}

type RpcResult struct {
	HooksConfig                shared.HooksConfig
	Buffers                    []*shared.ExtraBufferRPC
	BalancesRaw                []*big.Int
	BalancesLiveScaled18       []*big.Int
	TokenRates                 []*big.Int
	DecimalScalingFactors      []*big.Int
	AmplificationParameter     *big.Int
	StaticSwapFeePercentage    *big.Int
	AggregateSwapFeePercentage *big.Int
	IsVaultPaused              bool
	IsPoolPaused               bool
	IsPoolInRecoveryMode       bool
	BlockNumber                uint64
}
