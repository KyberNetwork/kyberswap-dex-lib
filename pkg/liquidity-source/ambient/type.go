package ambient

import (
	"math/big"
)

type IndexerPoolsResponse struct {
	Data []IndexerPool `json:"data"`
}

type IndexerPool struct {
	ChainID string `json:"chainId"`
	Base    string `json:"base"`
	Quote   string `json:"quote"`
	PoolIdx uint64 `json:"poolIdx"`
}

type StaticExtra struct {
	NativeToken string `json:"nT"`
	PoolIdx     uint64 `json:"pI"`
	SwapDex     string `json:"sD"`
	Base        string `json:"b"`
	Quote       string `json:"q"`
}

type Extra struct {
	State *TrackerExtra `json:"state"`
}

type Meta struct {
	SwapDex string   `json:"sD"`
	Base    string   `json:"b"`
	Quote   string   `json:"q"`
	PoolIdx *big.Int `json:"pI"`
}

type Gas struct {
	BaseGas          int64
	CrossInitTickGas int64
	PinSpillGas      int64
	KnockoutCrossGas int64
	ProtoFeeGas      int64
}
