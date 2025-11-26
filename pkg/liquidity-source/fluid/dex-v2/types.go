package dexv2

type Metadata struct {
	LastCreatedAtTimestamp int      `json:"lastCreatedAtTimestamp"`
	LastPoolIds            []string `json:"lastPoolIds"`
}

type SubgraphPool struct {
	ID          string `json:"id"`
	DexId       string `json:"dexId"`
	DexType     int    `json:"dexType"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	Fee         int    `json:"fee"`
	TickSpacing int    `json:"tickSpacing"`
	Controller  string `json:"controller"`
	CreatedAt   int    `json:"createdAt"`
}

type Extra struct {
	DexType     int     `json:"dexType"`
	Fee         int     `json:"fee"`
	TickSpacing int     `json:"tickSpacing"`
	Controller  string  `json:"controller,omitempty"`
	IsNative    [2]bool `json:"isNative"`
}
