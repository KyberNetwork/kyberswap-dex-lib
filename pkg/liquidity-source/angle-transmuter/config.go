package angletransmuter

type Config struct {
	DexID             string `json:"dexID"`
	ChainID           int    `json:"chainID"`
	Transmuter        string `json:"transmuter"`
	StableTokenMethod string `json:"stableTokenMethod"`
}
