package someswapv2

type StaticExtra struct {
	BaseFee uint32 `json:"baseFee"`
	WToken0 uint32 `json:"wToken0"`
	WToken1 uint32 `json:"wToken1"`
	Token0  string `json:"t0"`
	Token1  string `json:"t1"`
	Router  string `json:"router"`
}

type Extra struct {
	DynBps uint32 `json:"dynBps,omitempty"`
}

type PoolMeta struct {
	BaseFee  uint32 `json:"baseFee"`
	WToken0  uint32 `json:"wToken0"`
	WToken1  uint32 `json:"wToken1"`
	Router   string `json:"router"`
	TokenIn  string `json:"tokenIn"`
	TokenOut string `json:"tokenOut"`
}

type DynamicFeeResponse struct {
	Pool          string `json:"pool"`
	BaseFee       uint32 `json:"baseFee"`
	WToken0       string `json:"wToken0"`
	WToken1       string `json:"wToken1"`
	CurrentDynBps uint32 `json:"currentDynBps"`
	TotalFeeBps   uint32 `json:"totalFeeBps"`
	InBps         uint32 `json:"inBps"`
	OutBps        uint32 `json:"outBps"`
	Config        struct {
		Enabled   bool   `json:"enabled"`
		HalfLife  uint64 `json:"halfLife"`
		MaxCapBps uint32 `json:"maxCapBps"`
	} `json:"config"`
}

type GetPoolsResponse struct {
	Pools []APIPoolPair `json:"pools"`
}

type APIPoolPair struct {
	Token0 APIToken       `json:"token0"`
	Token1 APIToken       `json:"token1"`
	Pools  []APIPoolEntry `json:"pools"`
}

type APIPoolEntry struct {
	Backend   APIPoolBackend   `json:"backend"`
	FeeConfig APIPoolFeeConfig `json:"feeConfig"`
}

type APIPoolBackend struct {
	PairAddress string `json:"pair_address"`
}

type APIPoolFeeConfig struct {
	BaseFeeBps uint32 `json:"baseFeeBps"`
	WToken0In  string `json:"wToken0In"`
	WToken1In  string `json:"wToken1In"`
}

type APIToken struct {
	Address string `json:"address"`
}

type APIPool struct {
	PairAddress string
	Token0      APIToken
	Token1      APIToken
	BaseFee     uint32
	WToken0     uint32
	WToken1     uint32
}
