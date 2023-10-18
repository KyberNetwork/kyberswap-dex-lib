package sdk

import (
	"math/big"
)

type CurrencyAmount struct {
	currency           string
	amount             *big.Int
	amountMax          *big.Int
	newRes0            *big.Int
	newRes1            *big.Int
	newRes0Fic         *big.Int
	newRes1Fic         *big.Int
	newPriceAverage0   *big.Int
	newPriceAverage1   *big.Int
	userTradeTimestamp int64
}
