package crystalob

import (
	"errors"
	"math/big"

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
	gasBase     int64 = 195000
	gasPerLevel int64 = 22000
	gasAMM      int64 = 28000
)

var (
	// getPriceLevelsFromMid(distance, interval, max): every tick (clamped on-chain), exact prices.
	levelsDistance, levelsInterval, levelsMax = big.NewInt(1 << 62), big.NewInt(1), big.NewInt(maxLevels)
	maskU112                                  = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(uint256.NewInt(1), 112), 1)
	maskU41                                   = uint256.NewInt(1<<41 - 1)

	ErrInvalidToken     = errors.New("invalid token")
	ErrMarketInactive   = errors.New("market inactive")
	ErrInsufficientBook = errors.New("insufficient liquidity")
	ErrInvalidAmount    = errors.New("invalid amount")
	ErrAmountTooLarge   = errors.New("amount too large")
	ErrOverflow         = errors.New("overflow")
)
