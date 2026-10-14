package two

import (
	"github.com/holiman/uint256"
)

const nCoins = 2

var (
	DefaultGas  = Gas{Exchange: 220000}
	MinGamma    = uint256.NewInt(10_000_000_000)            // 1e10
	MaxGamma    = uint256.NewInt(20_000_000_000_000_000)    // 2e16
	AMultiplier = uint256.NewInt(10000)                     //
	MinA        = uint256.NewInt(4 * 10000 / 10)            // N_COINS**N_COINS * A_MULTIPLIER / 10
	MaxA        = uint256.NewInt(4 * 10000 * 100000)        // N_COINS**N_COINS * A_MULTIPLIER * 100000
	Precision   = uint256.NewInt(1_000_000_000_000_000_000) // 1e18
	PriceMask   = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(uint256.NewInt(1), 128), 1)
)

// Values (not pointers) so the arithmetic helpers can take them by value.
var (
	zero     uint256.Int
	one      = *uint256.NewInt(1)
	two      = *uint256.NewInt(2)
	hundred  = *uint256.NewInt(100)
	nCoinsU  = *uint256.NewInt(nCoins)
	bone     = *uint256.NewInt(1e18)
	twoBone  = *uint256.NewInt(2e18) // N_COINS * 10**18, also 2 * 10**18
	fourBone = *uint256.NewInt(4e18) // 2 * 10**18 * N_COINS
	halfBone = *uint256.NewInt(5e17)

	tenPow5  = *uint256.NewInt(1e5)
	tenPow6  = *uint256.NewInt(1e6)
	tenPow9  = *uint256.NewInt(1e9)
	tenPow10 = *uint256.NewInt(1e10)
	tenPow11 = *uint256.NewInt(1e11)
	tenPow14 = *uint256.NewInt(1e14)
	tenPow16 = *uint256.NewInt(1e16)
	tenPow17 = *uint256.NewInt(1e17)
	tenPow20 = *uint256.MustFromDecimal("100000000000000000000")
	tenPow33 = *uint256.MustFromDecimal("1000000000000000000000000000000000")
)
