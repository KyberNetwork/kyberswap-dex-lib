package parallelprotocol

type Config struct {
	DexID          string `json:"dexID"`
	StableToken    string `json:"st"`
	StableDecimals uint8  `json:"sd"`
	Parallelizer     string `json:"parallelizer"`
	ChainID        int    `json:"chainID"`
}
