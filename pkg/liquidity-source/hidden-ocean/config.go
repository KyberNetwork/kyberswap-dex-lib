package hiddenocean

type Config struct {
	DexID           string `json:"dexId"`
	RegistryAddress string `json:"registryAddress"`
	NewPoolLimit    int    `json:"newPoolLimit"`
}
