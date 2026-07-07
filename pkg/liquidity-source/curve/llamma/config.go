package llamma

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	BorrowedToken  string `json:"borrowedToken"`
	MaxBandLimit   int64  `json:"maxBandLimit"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
