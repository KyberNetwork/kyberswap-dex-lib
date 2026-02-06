package kipseliprop

type Config struct {
	DexID         string `json:"dexID"`
	ChainID       int    `json:"chainID"`
	LensAddress   string `json:"lensAddress"`
	RouterAddress string `json:"routerAddress"`
	Buffer        int64  `json:"buffer"`
}
