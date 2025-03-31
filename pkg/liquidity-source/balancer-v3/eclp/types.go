package eclp

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
)

type RpcResult struct {
	HooksConfig                shared.HooksConfig
	Buffers                    []*shared.ExtraBufferRPC
	BalancesRaw                []*big.Int
	BalancesLiveScaled18       []*big.Int
	TokenRates                 []*big.Int
	DecimalScalingFactors      []*big.Int
	StaticSwapFeePercentage    *big.Int
	AggregateSwapFeePercentage *big.Int
	IsVaultPaused              bool
	IsPoolPaused               bool
	IsPoolInRecoveryMode       bool
	BlockNumber                uint64
	ECLPParams                 ECLPParamsResp
}

type Extra struct {
	HooksConfig                shared.HooksConfig    `json:"hooksConfig"`
	StaticSwapFeePercentage    *uint256.Int          `json:"staticSwapFeePercentage"`
	AggregateSwapFeePercentage *uint256.Int          `json:"aggregateSwapFeePercentage"`
	ECLPParams                 ECLPParams            `json:"eclpParams"`
	BalancesLiveScaled18       []*uint256.Int        `json:"balancesLiveScaled18"`
	DecimalScalingFactors      []*uint256.Int        `json:"decimalScalingFactors"`
	TokenRates                 []*uint256.Int        `json:"tokenRates"`
	Buffers                    []*shared.ExtraBuffer `json:"buffers"`
	IsVaultPaused              bool                  `json:"isVaultPaused,omitempty"`
	IsPoolPaused               bool                  `json:"isPoolPaused,omitempty"`
	IsPoolInRecoveryMode       bool                  `json:"isPoolInRecoveryMode,omitempty"`
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

type ECLPParams struct {
	Params struct {
		Alpha  *int256.Int
		Beta   *int256.Int
		C      *int256.Int
		S      *int256.Int
		Lambda *int256.Int
	}

	D struct {
		TauAlpha struct {
			X *int256.Int
			Y *int256.Int
		}
		TauBeta struct {
			X *int256.Int
			Y *int256.Int
		}
		U   *int256.Int
		V   *int256.Int
		W   *int256.Int
		Z   *int256.Int
		DSq *int256.Int
	}
}

func (p *ECLPParamsResp) toInt256() ECLPParams {
	var result ECLPParams

	result.Params.Alpha = int256.MustFromBig(p.Params.Alpha)
	result.Params.Beta = int256.MustFromBig(p.Params.Beta)
	result.Params.C = int256.MustFromBig(p.Params.C)
	result.Params.S = int256.MustFromBig(p.Params.S)
	result.Params.Lambda = int256.MustFromBig(p.Params.Lambda)

	result.D.TauAlpha.X = int256.MustFromBig(p.D.TauAlpha.X)
	result.D.TauAlpha.Y = int256.MustFromBig(p.D.TauAlpha.Y)
	result.D.TauBeta.X = int256.MustFromBig(p.D.TauBeta.X)
	result.D.TauBeta.Y = int256.MustFromBig(p.D.TauBeta.Y)
	result.D.U = int256.MustFromBig(p.D.U)
	result.D.V = int256.MustFromBig(p.D.V)
	result.D.W = int256.MustFromBig(p.D.W)
	result.D.Z = int256.MustFromBig(p.D.Z)
	result.D.DSq = int256.MustFromBig(p.D.DSq)

	return result
}
