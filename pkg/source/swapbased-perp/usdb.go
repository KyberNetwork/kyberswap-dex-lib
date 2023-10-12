package swapbasedperp

import (
	"math/big"
)

type USDB struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	usdbMethodTotalSupply = "totalSupply"
)
