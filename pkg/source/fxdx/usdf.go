package fxdx

import "math/big"

type USDF struct {
	Address     string   `json:"address"`
	TotalSupply *big.Int `json:"totalSupply"`
}

const (
	usdfMethodTotalSupply = "totalSupply"
)
