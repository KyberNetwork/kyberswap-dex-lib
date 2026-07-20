package pamm

type Config struct {
	DexID                  string      `json:"dexID"`
	ChainID                int         `json:"chainId"`
	RouterAddress          string      `json:"routerAddress"`
	LensAddress            string      `json:"lensAddress,omitempty"`
	PriorityUpdateRegistry string      `json:"priorityUpdateRegistry,omitempty"`
	Multicall3Address      string      `json:"multicall3Address,omitempty"`
	PositionCapAddress     string      `json:"positionCapAddress,omitempty"`
	Titan                  TitanConfig `json:"titan,omitempty"`
}
