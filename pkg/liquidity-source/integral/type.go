package integral

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type IntegralPair struct {
	Reserve [2]*big.Int
	PairFee [2]*big.Int

	MintFee *big.Int
	BurnFee *big.Int
	SwapFee *big.Int
	Oracle  common.Address
}

type PairFee struct {
	Fee0 *big.Int
	Fee1 *big.Int
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
