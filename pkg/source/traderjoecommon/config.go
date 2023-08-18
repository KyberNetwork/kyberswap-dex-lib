package traderjoecommon

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	RouterAddress  string `json:"routerAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
