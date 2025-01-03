package syncswapv2

type Config struct {
	DexID         string   `json:"dexID"`
	MasterAddress []string `json:"masterAddress"`
	NewPoolLimit  int      `json:"newPoolLimit"`
}
