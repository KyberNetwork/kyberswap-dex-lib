package camelot

type Config struct {
	DexID          string `json:"-"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   uint   `json:"newPoolLimit"`
}
