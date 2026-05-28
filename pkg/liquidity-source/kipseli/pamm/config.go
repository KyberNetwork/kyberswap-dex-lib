package pamm

// LensAddress also serves as the on-chain pricing state — Titan publishes
// storage overrides here that both the off-chain read API and the engine read.
type Config struct {
	DexID         string      `json:"dexID"`
	ChainID       int         `json:"chainId"`
	LensAddress   string      `json:"lensAddress"`
	RouterAddress string      `json:"routerAddress"`
	Titan         TitanConfig `json:"titan,omitempty"`
}
