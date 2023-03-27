package metavault

import (
	"math/big"
)

type USDM struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	USDMMethodTotalSupply = "totalSupply"
)
