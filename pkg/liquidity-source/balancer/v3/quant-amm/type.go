package quantamm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
)

type StaticExtra struct {
	*shared.StaticExtra
	MaxTradeSizeRatio *uint256.Int `json:"mxTSR,omitempty"`
}

type Extra struct {
	*shared.Extra
	Weights         []*uint256.Int `json:"w,omitempty"`
	Multipliers     []*uint256.Int `json:"m,omitempty"`
	LastUpdateTime  uint64         `json:"u,omitempty"`
	LastInteropTime uint64         `json:"i,omitempty"`
}

type RpcResult struct {
	shared.RpcResult
	DynamicDataRpc
	ImmutableDataRpc
}

type DynamicDataRpc struct {
	DynamicData struct {
		BalancesLiveScaled18            []*big.Int
		TokenRates                      []*big.Int
		TotalSupply                     *big.Int
		IsPoolInitialized               bool
		IsPoolPaused                    bool
		IsPoolInRecoveryMode            bool
		FirstFourWeightsAndMultipliers  []*big.Int
		SecondFourWeightsAndMultipliers []*big.Int
		LastUpdateTime                  uint64
		LastInteropTime                 uint64
	}
}

type ImmutableDataRpc struct {
	ImmutableData struct {
		Tokens                   []common.Address
		OracleStalenessThreshold *big.Int
		PoolRegistry             *big.Int
		RuleParameters           [][]*big.Int
		Lambda                   []uint64
		EpsilonMax               uint64
		AbsoluteWeightGuardRail  uint64
		UpdateInterval           uint64
		MaxTradeSizeRatio        *big.Int
	}
}
