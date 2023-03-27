package madmex

import (
	"math/big"
)

type USDG struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	USDGMethodTotalSupply = "totalSupply"
)
