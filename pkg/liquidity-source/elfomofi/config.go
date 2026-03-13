package elfomofi

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	Buffer         int64  `json:"buffer"` // in bps
}
