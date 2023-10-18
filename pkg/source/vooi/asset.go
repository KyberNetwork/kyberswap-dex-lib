package vooi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Asset struct {
	Cash        *big.Int       `json:"cash"`
	Liability   *big.Int       `json:"liability"`
	MaxSupply   *big.Int       `json:"maxSupply"`
	TotalSupply *big.Int       `json:"totalSupply"`
	Decimals    uint8          `json:"decimals"`
	Token       common.Address `json:"token"`
	Active      bool           `json:"active"`
}
