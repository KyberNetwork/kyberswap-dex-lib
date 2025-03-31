package eclp

import (
	"math/big"

	"github.com/KyberNetwork/int256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type Extra struct {
	*shared.Extra
	ECLPParams ECLPParams `json:"eclp"`
}

type ECLPParams struct {
	Params struct {
		Alpha  *int256.Int `json:"a,omitempty"`
		Beta   *int256.Int `json:"b,omitempty"`
		C      *int256.Int `json:"c,omitempty"`
		S      *int256.Int `json:"s,omitempty"`
		Lambda *int256.Int `json:"l,omitempty"`
	} `json:"p,omitempty"`

	D struct {
		TauAlpha struct {
			X *int256.Int `json:"x,omitempty"`
			Y *int256.Int `json:"y,omitempty"`
		} `json:"tA"`
		TauBeta struct {
			X *int256.Int `json:"x,omitempty"`
			Y *int256.Int `json:"y,omitempty"`
		} `json:"tB"`
		U   *int256.Int `json:"u,omitempty"`
		V   *int256.Int `json:"v,omitempty"`
		W   *int256.Int `json:"w,omitempty"`
		Z   *int256.Int `json:"z,omitempty"`
		DSq *int256.Int `json:"DSq,omitempty"`
	} `json:"d,omitempty"`
}

type ECLPParamsRpc struct {
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

type RpcResult struct {
	shared.RpcResult
	ECLPParamsRpc
}

func (p *ECLPParamsRpc) toInt256() ECLPParams {
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
