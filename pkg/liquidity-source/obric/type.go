package obric

import "math/big"

type PoolState struct {
	DecimalsX       uint8
	DecimalsY       uint8
	MultYBase       *big.Int
	ReserveX        *big.Int
	ReserveY        *big.Int
	CurrentXK       *big.Int
	PreK            *big.Int
	FeeMillionth    uint64
	PriceMaxAge     *big.Int
	PriceUpdateTime *big.Int
	IsLocked        bool
	Diff            *big.Int
	Enable          bool
}

type StaticExtra struct {
	PoolId    int    `json:"pId"`
	MultYBase string `json:"mYB"`

	DependenciesStored bool `json:"ds,omitempty"`
}

type Extra struct {
	ReserveX        string `json:"rX"`
	ReserveY        string `json:"rY"`
	CurrentXK       string `json:"currentXK"`
	PreK            string `json:"preK"`
	FeeMillionth    uint64 `json:"feeMillionth"`
	PriceMaxAge     uint64 `json:"pMaxAge"`
	PriceUpdateTime uint64 `json:"priceUpdateTime"`
	IsLocked        bool   `json:"IsLocked,omitempty"`
	Enable          bool   `json:"enable,omitempty"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"bN"`
	IsXtoY      bool   `json:"isXtoY,omitempty"`
}
