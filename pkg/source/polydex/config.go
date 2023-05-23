package polydex

type Config struct {
	DexID          string `json:"-"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
