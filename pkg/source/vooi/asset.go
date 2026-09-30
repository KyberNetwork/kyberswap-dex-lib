package vooi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
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

type assetRPC struct {
	Cash        *uint256.Int
	Liability   *uint256.Int
	MaxSupply   *uint256.Int
	TotalSupply *uint256.Int
	Decimals    uint8
	Token       common.Address
	Active      bool
}

func (a assetRPC) toAsset() Asset {
	return Asset{
		Cash:        u256ToBig(a.Cash),
		Liability:   u256ToBig(a.Liability),
		MaxSupply:   u256ToBig(a.MaxSupply),
		TotalSupply: u256ToBig(a.TotalSupply),
		Decimals:    a.Decimals,
		Token:       a.Token,
		Active:      a.Active,
	}
}
