package bouncetech

type Config struct {
	DexID                string `json:"dexID"`
	FactoryAddress       string `json:"factoryAddress"`
	GlobalStorageAddress string `json:"globalStorageAddress"`
	NewPoolLimit         int    `json:"newPoolLimit"`
}
