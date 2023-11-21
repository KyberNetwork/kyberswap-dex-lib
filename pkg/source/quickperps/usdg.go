package quickperps

import (
	"math/big"
)

type USDQ struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	usdqMethodTotalSupply = "totalSupply"
)
