package elfomofi

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	HelperAddress  string `json:"helperAddress"` // optional override of the known chain's ElfomoFiHelper
	Buffer         int64  `json:"buffer"`        // in bps
}
