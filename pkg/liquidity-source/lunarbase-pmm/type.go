package lunarbase

import "github.com/holiman/uint256"

type Metadata struct {
	Initialized bool `json:"initialized"`
}

type Extra struct {
	PX96               *uint256.Int `json:"pX96"`
	Fee                uint64       `json:"fee"`
	LatestUpdateBlock  uint64       `json:"latestUpdateBlock"`
	Paused             bool         `json:"paused"`
	BlockDelay         uint64       `json:"blockDelay"`
	ConcentrationK     uint32       `json:"concentrationK"`
	ConcentrationAlpha uint8        `json:"concentrationAlpha"`
}

type StaticExtra struct {
	PeripheryAddress string `json:"peripheryAddress"`
	Permit2Address   string `json:"permit2Address"`
	RawTokenX        string `json:"rawTokenX"`
	RawTokenY        string `json:"rawTokenY"`
	WrappedNative    string `json:"wrappedNative"`
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	RouterAddress   string `json:"routerAddress"`
	Permit2Address  string `json:"permit2Address"`
	ApprovalAddress string `json:"approvalAddress"`
}

type PoolParams struct {
	SqrtPriceX96   *uint256.Int
	FeeQ48         uint64
	ReserveX       *uint256.Int
	ReserveY       *uint256.Int
	ConcentrationK uint32
}

type QuoteResult struct {
	AmountOut     *uint256.Int
	SqrtPriceNext *uint256.Int
	Fee           *uint256.Int
}
