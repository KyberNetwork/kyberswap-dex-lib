package integral

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type IntegralPair struct {
	Reserve [2]*uint256.Int
	PairFee [2]*uint256.Int

	MintFee *uint256.Int
	BurnFee *uint256.Int
	SwapFee *uint256.Int
	Oracle  common.Address
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
