package velocorev2stable

type Config struct {
	DexID           string `json:"-"`
	RegistryAddress string `json:"registryAddress"`
	NewPoolLimit    int    `json:"newPoolLimit"`
	LensAddress     string `json:"lensAddress"`
}
