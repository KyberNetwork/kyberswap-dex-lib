package someswap

import "math/big"

type ReserveData struct {
	Reserve0           *big.Int `abi:"_r0"`
	Reserve1           *big.Int `abi:"_r1"`
	BlockTimestampLast uint32   `abi:"ts"`
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil
}

type Metadata struct {
	Offset int `json:"offset"`
}

type StaticExtra struct {
	BaseFeeBps    string `json:"baseFeeBps"`
	DynamicFeeBps string `json:"dynamicFeeBps"`
	WToken0In     string `json:"wToken0In"`
	WToken1In     string `json:"wToken1In"`
}

type PoolMeta struct {
	BaseFeeBps    string `json:"baseFeeBps"`
	DynamicFeeBps string `json:"dynamicFeeBps"`
	WToken0In     string `json:"wToken0In"`
	WToken1In     string `json:"wToken1In"`
}
