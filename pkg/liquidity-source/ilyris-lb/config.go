package ilyrislb

// Config is the pool-service dex config.
type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	LensAddress    string `json:"lensAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
