package someswapv2

import "math/big"

type ReserveData struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil
}

type Metadata struct {
	Offset int `json:"offset"`
}

type BaseFeeConfig struct {
	BaseFee uint32 `json:"baseFee"`
	WToken0 uint32 `json:"wToken0"`
	WToken1 uint32 `json:"wToken1"`
}

type StaticExtra struct {
	BaseFee uint32 `json:"baseFee"`
	WToken0 uint32 `json:"wToken0"`
	WToken1 uint32 `json:"wToken1"`
}

type PoolMeta struct {
	BaseFee uint32 `json:"baseFee"`
	WToken0 uint32 `json:"wToken0"`
	WToken1 uint32 `json:"wToken1"`
}
