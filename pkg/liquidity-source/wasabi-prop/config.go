package wasabiprop

type Config struct {
	DexID          string `json:"dexID"`
	ChainID        int    `json:"chainID"`
	FactoryAddress string `json:"factoryAddress"`
	RouterAddress  string `json:"routerAddress"`
	Buffer         int64  `json:"buffer"`
}
