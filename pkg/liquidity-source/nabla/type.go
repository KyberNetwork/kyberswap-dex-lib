package nabla

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
)

type NablaPoolMeta struct {
	CurveBeta   *int256.Int `json:"curveBeta"`
	CurveC      *int256.Int `json:"curveC"`
	BackstopFee *int256.Int `json:"backstopFee"`
	ProtocolFee *int256.Int `json:"protocolFee"`
	LpFee       *int256.Int `json:"lpFee"`
}

type NablaPoolState struct {
	Reserve             *int256.Int `json:"reserve"`
	ReserveWithSlippage *int256.Int `json:"reserveWithSlippage"`
	TotalLiabilities    *int256.Int `json:"totalLiabilities"`
	Price               *int256.Int `json:"price"`
}

type NablaPool struct {
	Address common.Address `json:"address"`
	Curve   common.Address `json:"curve"`
	Meta    NablaPoolMeta  `json:"meta"`
	State   NablaPoolState `json:"state"`
}

type SwapFees struct {
	LpFee       *big.Int
	BackstopFee *big.Int
	ProtocolFee *big.Int
}

type Params struct {
	Beta *big.Int
	C    *big.Int
}

type Extra struct {
	Pools              []NablaPool `json:"pools"`
	DependenciesStored bool        `json:"ds,omitempty"`
}

type Meta struct {
	BlockNumber uint64 `json:"bN"`
}

type SwapInfo struct {
	frPoolNewState NablaPoolState
	toPoolNewState NablaPoolState
}
