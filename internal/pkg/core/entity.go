package core

import "math/big"

type GasOption struct {
	GasFeeInclude bool
	Price         *big.Float
	TokenPrice    float64
}

type BestPathOption struct {
	MaxHops  uint32
	MaxPools uint32
	MaxPaths uint32
}

type BestRouteOption struct {
	MaxHops    uint32
	MaxPools   uint32
	MaxPaths   uint32
	MinPartUsd float64
	SaveGas    bool
	Gas        GasOption
}
