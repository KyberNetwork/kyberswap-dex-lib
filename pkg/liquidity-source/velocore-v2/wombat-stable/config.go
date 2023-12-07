package wombatstable

type Config struct {
	DexID           string `json:"dexID"`
	VaultAddress    string `json:"vaultAddress"`
	RegistryAddress string `json:"registryAddress"`
	NewPoolLimit    int    `json:"newPoolLimit"`
	LensAddress     string `json:"lensAddress"`
}
