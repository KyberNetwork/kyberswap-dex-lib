package ilyris

// Config is what the aggregator service reads from its dex config file.
type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	LensAddress    string `json:"lensAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
