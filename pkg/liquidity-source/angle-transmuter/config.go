package angletransmuter

type Config struct {
	DexID          string `json:"dexID"`
	StableToken    string `json:"st"`
	StableDecimals uint8  `json:"sd"`
	Transmuter     string `json:"transmuter"`
	Pyth           string `json:"pyth"`
	ChainID        int    `json:"chainID"`
}
