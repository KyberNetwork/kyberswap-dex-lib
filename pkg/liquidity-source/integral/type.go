package integral

import (
	"math/big"

	"github.com/holiman/uint256"
)

type IntegralPair struct {
	DecimalsConverter *big.Int
	SwapFee           *uint256.Int
	AveragePrice      *uint256.Int
}

type PriceInfo struct {
	PriceAccumulator *big.Int
	PriceTimestamp   *big.Int
}

type PairFee struct {
	Fee0 *uint256.Int
	Fee1 *uint256.Int
}

type GetAmountParameters struct {
	amount          *big.Int
	reserveIn       *big.Int
	reserveOut      *big.Int
	priceAverageIn  *big.Int
	priceAverageOut *big.Int
	feesLP          *big.Int
	feesPool        *big.Int
	feesBase        *big.Int
}

type GetAmountResult struct {
	amountOut            *big.Int
	newReserveIn         *big.Int
	newReserveOut        *big.Int
	newFictiveReserveIn  *big.Int
	newFictiveReserveOut *big.Int
}

type SwapInfo struct {
	newReserveIn  *big.Int
	newReserveOut *big.Int
}
