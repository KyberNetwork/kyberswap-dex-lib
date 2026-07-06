package prop

type Config struct {
	DexID         string `json:"dexID"`
	ChainID       int    `json:"chainId"`
	RouterAddress string `json:"routerAddress"`
	Buffer        int64  `json:"buffer"`
}
