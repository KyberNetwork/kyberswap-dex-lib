package smardex

import (
	"math/big"
)

type Gas struct {
	Swap int64
}

type SmardexPair struct {

	// smardex pair fees numerators, denominator is 1_000_000
	PairFee PairFee

	// smardex new fictive reserves
	FictiveReserve FictiveReserve

	// moving average on the price
	PriceAverage PriceAverage

	// fee for FEE_POOL
	FeeToAmount FeeToAmount
}

type PairFee struct {
	FeesLP   *big.Int
	FeesPool *big.Int
	FeesBase *big.Int
}

type FictiveReserve struct {
	FictiveReserve0 *big.Int
	FictiveReserve1 *big.Int
}

type PriceAverage struct {
	PriceAverage0             *big.Int
	PriceAverage1             *big.Int
	PriceAverageLastTimestamp *big.Int
}

type FeeToAmount struct {
	Fees0 *big.Int
	Fees1 *big.Int
}

type Reserve struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

type GetAmountParameters struct {
	amount            *big.Int
	reserveIn         *big.Int
	reserveOut        *big.Int
	fictiveReserveIn  *big.Int
	fictiveReserveOut *big.Int
	priceAverageIn    *big.Int
	priceAverageOut   *big.Int
	feesLP            *big.Int
	feesPool          *big.Int
	feesBase          *big.Int
}

type GetAmountResult struct {
	amountOut            *big.Int
	newReserveIn         *big.Int
	newReserveOut        *big.Int
	newFictiveReserveIn  *big.Int
	newFictiveReserveOut *big.Int
}

type SwapInfo struct {
	newReserveIn              *big.Int
	newReserveOut             *big.Int
	newFictiveReserveIn       *big.Int
	newFictiveReserveOut      *big.Int
	newPriceAverageIn         *big.Int
	newPriceAverageOut        *big.Int
	priceAverageLastTimestamp *big.Int
	feeToAmount0              *big.Int
	feeToAmount1              *big.Int
}
