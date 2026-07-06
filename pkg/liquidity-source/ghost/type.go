package ghost

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolItem struct {
	ID          string             `json:"id"`
	Type        string             `json:"type"`
	Tokens      []entity.PoolToken `json:"tokens"`
	StaticExtra StaticExtra        `json:"staticExtra"`
}

// DirectionStatic holds the immutable router/scale info needed to swap in one direction.
type DirectionStatic struct {
	SourceRouter     string `json:"src"`
	TargetRouter     string `json:"dst"`
	LocalDomain      uint32 `json:"dom"`
	ScaleNumerator   string `json:"sNum"`
	ScaleDenominator string `json:"sDen"`
}

// StaticExtra holds both swap directions for the pair: ZeroToOne is tokens[0] -> tokens[1],
// OneToZero is tokens[1] -> tokens[0] (naming matches the zeroForOne convention used elsewhere
// in this repo, e.g. Uniswap-style pools). Each direction is an independent on-chain call (its
// own sourceRouter, fee curve, and targetRouter-held reserve) — the two directions share no
// state, they just happen to connect the same token pair.
type StaticExtra struct {
	ZeroToOne DirectionStatic `json:"z2o"`
	OneToZero DirectionStatic `json:"o2z"`
}

// DirectionExtra holds the mutable fee-curve/reserve state for one direction, refreshed by the
// tracker.
type DirectionExtra struct {
	MaxFee     *big.Int `json:"maxFee"`
	HalfAmount *big.Int `json:"halfAmt"`
	Reserve    *big.Int `json:"reserve"`
}

type Extra struct {
	ZeroToOne DirectionExtra `json:"z2o"`
	OneToZero DirectionExtra `json:"o2z"`
}

type PoolMeta struct {
	SourceRouter string `json:"sourceRouter"`
	TargetRouter string `json:"targetRouter"`
}

// SwapInfo carries the feeBps executeGhost needs to recover this trade's principal from
// amountIn on-chain (see totalFeeBps in pool_simulator.go).
type SwapInfo struct {
	TotalFeeBps int64 `json:"totalFeeBps"`
}
