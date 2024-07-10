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
	Base    string `json:"base"`
	Quote   string `json:"quote"`
	PoolIdx string `json:"pool_idx"`
}

type Extra struct {
	SqrtPriceX64 string `json:"sqrtPriceX64"`
	Liquidity    string `json:"liquidity"`
}
