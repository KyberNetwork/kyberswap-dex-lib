package synthetix

type Config struct {
	DexID     string    `json:"-"`
	ChainID   uint      `json:"chainId"`
	Addresses Addresses `json:"addresses"`
}
