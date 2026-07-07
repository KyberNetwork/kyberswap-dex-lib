package smardex

import (
	"math/big"

	"github.com/holiman/uint256"
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

type PairFeeResult struct {
	FeesLP   *big.Int
	FeesPool *big.Int
	FeesBase *big.Int
}

type PairFee struct {
	FeesLP   *uint256.Int
	FeesPool *uint256.Int
	FeesBase *uint256.Int
}

type FictiveReserveResult struct {
	FictiveReserve0 *big.Int
	FictiveReserve1 *big.Int
}

type FictiveReserve struct {
	FictiveReserve0 *uint256.Int
	FictiveReserve1 *uint256.Int
}

type PriceAverageResult struct {
	PriceAverage0             *big.Int
	PriceAverage1             *big.Int
	PriceAverageLastTimestamp *big.Int
}

type PriceAverage struct {
	PriceAverage0             *uint256.Int
	PriceAverage1             *uint256.Int
	PriceAverageLastTimestamp *uint256.Int
}

type FeeToAmountResult struct {
	Fees0 *big.Int
	Fees1 *big.Int
}

type FeeToAmount struct {
	Fees0 *uint256.Int
	Fees1 *uint256.Int
}

type Reserve struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

type GetAmountParameters struct {
	amount            *uint256.Int
	reserveIn         *uint256.Int
	reserveOut        *uint256.Int
	fictiveReserveIn  *uint256.Int
	fictiveReserveOut *uint256.Int
	priceAverageIn    *uint256.Int
	priceAverageOut   *uint256.Int
	feesLP            *uint256.Int
	feesPool          *uint256.Int
	feesBase          *uint256.Int
}

type GetAmountResult struct {
	amountOut            *uint256.Int
	newReserveIn         *uint256.Int
	newReserveOut        *uint256.Int
	newFictiveReserveIn  *uint256.Int
	newFictiveReserveOut *uint256.Int
}

type SwapInfo struct {
	newReserveIn              *uint256.Int
	newReserveOut             *uint256.Int
	newFictiveReserveIn       *uint256.Int
	newFictiveReserveOut      *uint256.Int
	newPriceAverageIn         *uint256.Int
	newPriceAverageOut        *uint256.Int
	priceAverageLastTimestamp *uint256.Int
	feeToAmount0              *uint256.Int
	feeToAmount1              *uint256.Int
}
