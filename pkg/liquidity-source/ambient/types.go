package ambient

const (
	DexType    = "ambient"
	fetchLimit = 1000
)

type FetchPoolsResponse struct {
	Data Data `json:"data"`
}

type Pool struct {
	ID          string `json:"id"`
	BlockCreate string `json:"blockCreate"`
	TimeCreate  uint64 `json:"timeCreate,string"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	PoolIdx     string `json:"poolIdx"`
}

type Data struct {
	Pools []Pool `json:"pools"`
}

type PoolListUpdaterMetadata struct {
	LastCreateTime uint64 `json:"lastCreateTime"`
}
