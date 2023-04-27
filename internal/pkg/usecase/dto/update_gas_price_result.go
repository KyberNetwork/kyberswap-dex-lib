package dto

import (
	"math/big"
)

type UpdateGasPriceResult struct {
	SuggestedGasPrice *big.Int
}
