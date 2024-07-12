package ambient

type FetchPoolsResponse struct {
	Pools []Pool `json:"pools"`
}

type Pool struct {
	ID          string `json:"id"`
	BlockCreate string `json:"blockCreate"`
	TimeCreate  uint64 `json:"timeCreate,string"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	PoolIdx     string `json:"poolIdx"`
}

type PoolListUpdaterMetadata struct {
	LastCreateTime uint64 `json:"lastCreateTime"`
}

type StaticExtra struct {
	// Base token, 0x0 if native token
	Base string `json:"base"`
	// Quote token
	Quote string `json:"quote"`
	// The index or discriminator of the pool in a logical list of pools with (base, quote) tokens. Default is "420".
	PoolIdx string `json:"poolIdx"`
	// The CrocSwapDex.sol contract address
	SwapAddress string `json:"swapAddress"`
}

type Extra struct {
	SqrtPriceX64 string `json:"sqrtPriceX64"`
	Liquidity    string `json:"liquidity"`
}
