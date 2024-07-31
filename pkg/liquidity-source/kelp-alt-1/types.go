package rsethalt1

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	// MinAmountToDeposit  *big.Int            `json:"minAmountToDeposit"`
	// TotalDepositByAsset map[string]*big.Int `json:"totalDepositByAsset"`
	// DepositLimitByAsset map[string]*big.Int `json:"depositLimitByAsset"`
	PriceByAsset map[string]*big.Int `json:"priceByAsset"`
	// RSETHPrice          *big.Int            `json:"rsETHPrice"`
	FeeBps *big.Int `json:"feeBps"`

	supportedTokens []*entity.PoolToken
}
