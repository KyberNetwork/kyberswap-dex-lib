package maverick

import "math/big"

type Metadata struct {
	LastCreateTime *big.Int
}

type SubgraphPool struct {
	ID          string        `json:"id"`
	Fee         float64       `json:"fee"`
	TickSpacing *big.Int      `json:"tickSpacing"`
	Timestamp   *big.Int      `json:"timestamp"`
	TokenA      SubgraphToken `json:"tokenA"`
	TokenB      SubgraphToken `json:"tokenB"`
}

type SubgraphToken struct {
	ID       string `json:"id"`
	Decimals uint8  `json:"decimals"`
}

type StaticExtra struct {
	TickSpacing *big.Int `json:"tickSpacing"`
}
