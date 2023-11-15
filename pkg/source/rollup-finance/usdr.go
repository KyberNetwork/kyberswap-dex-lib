package rollupfinance

import (
	"math/big"
)

type USDR struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	usdrMethodTotalSupply = "totalSupply"
)
