package smardex

import (
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	poolpkg.Pool
	SmardexPair
	gas Gas
}

type Gas struct {
	Swap int64
}

type SmardexPair struct {

	// smardex pair fees numerators, denominator is 1_000_000
	pairFee PairFee

	// smardex new fictive reserves
	fictiveReserve FictiveReseerve

	// moving average on the price
	priceAverage PriceAverage

	// fee for FEE_POOL
	feeToAmount FeeToAmount

	// access through balanceOf of ERC20 token
	reserve Reserve
}

type PairFee struct {
	feesLP   *big.Int
	feesPool *big.Int
}

type FictiveReseerve struct {
	fictiveReserve0 *big.Int
	fictiveReserve1 *big.Int
}

type PriceAverage struct {
	priceAverage0             *big.Int
	priceAverage1             *big.Int
	priceAverageLastTimestamp int64
}

type FeeToAmount struct {
	feeToAmount0 *big.Int
	feeToAmount1 *big.Int
}

type Reserve struct {
	reserve0 *big.Int
	reserve1 *big.Int
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
}

type GetAmountResult struct {
	amountOut            *big.Int
	newReserveIn         *big.Int
	newReserveOut        *big.Int
	newFictiveReserveIn  *big.Int
	newFictiveReserveOut *big.Int
}
