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

type StaticExtra struct {
	SourceRouter     string `json:"sourceRouter"`
	TargetRouter     string `json:"targetRouter"`
	LocalDomain      uint32 `json:"localDomain"`
	ScaleNumerator   string `json:"scaleNumerator"`
	ScaleDenominator string `json:"scaleDenominator"`
}

type Extra struct {
	MaxFee     *big.Int `json:"maxFee"`
	HalfAmount *big.Int `json:"halfAmount"`
	Reserve    *big.Int `json:"reserve"`
}

type PoolMeta struct {
	SourceRouter string `json:"sourceRouter"`
	TargetRouter string `json:"targetRouter"`
}
