package crystalob

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "crystal-ob"

	maxLevels      = 64 // per side, from getPriceLevelsFromMid
	maxOrderRounds = 8  // linked-list hops resolved per multi-order level
	listBatchSize  = 50

	ammFeeDefault = 9975 // Crystal.DEFAULT_AMM_FEE
	ammFeeFour    = 9900 // Crystal.FOUR_AMM_FEE, marketType 4 (graduated launchpad)

	// executeCrystal gas from Tenderly sims on Monad (1/4/6 book levels, AMM only), excluding the
	// executor's one-time Crystal.registerUser (~62k).
	gasBase     = 195000
	gasPerLevel = 22000
	gasAMM      = 28000
)

var (
	levelsDistance = uint256.NewInt(1 << 62) // ticks; the contract clamps it to the book bounds
	maskU112       = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(uint256.NewInt(1), 112), 1)
	maskU41        = uint256.NewInt(1<<41 - 1)

	ErrInvalidToken     = errors.New("invalid token")
	ErrMarketInactive   = errors.New("market inactive")
	ErrInsufficientBook = errors.New("insufficient liquidity")
	ErrInvalidAmount    = errors.New("invalid amount")
	ErrAmountTooLarge   = errors.New("amount too large")
	ErrOverflow         = errors.New("overflow")
)
