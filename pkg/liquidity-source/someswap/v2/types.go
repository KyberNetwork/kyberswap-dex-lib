package someswapv2

type StaticExtra struct {
	BaseFee      uint32 `json:"baseFee"`
	WToken0      uint32 `json:"wToken0"`
	WToken1      uint32 `json:"wToken1"`
	NativeToken0 bool   `json:"n0"`
	NativeToken1 bool   `json:"n1"`
}

type PoolMeta struct {
	BaseFee uint32 `json:"baseFee"`
	WToken0 uint32 `json:"wToken0"`
	WToken1 uint32 `json:"wToken1"`
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
