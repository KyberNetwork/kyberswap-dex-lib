package sd59x18

import "math/big"

var (
	unit = big.NewInt(1e18)
)

var (
	uUnit = big.NewInt(1e18)
)

var (
	// The minimum value an SD59x18 number can have.
	uMinSD59x18, _ = new(big.Int).SetString("-57896044618658097711785492504343953926634992332820282019728792003956564819968", 10)
)

var (
	uMaxSD59x18, _ = new(big.Int).SetString("57896044618658097711785492504343953926634992332820282019728792003956564819967", 10)
)

var (
	zeroBI = big.NewInt(0)
)

var (
	// 1e36
	uUnitSquared, _ = new(big.Int).SetString("1000000000000000000000000000000000000", 10)
)

var (
	// 0.5e18
	uHalfUnit, _ = new(big.Int).SetString("500000000000000000", 10)
)

var (
	// 192e18 - 1
	uExp2MaxInput, _ = new(big.Int).SetString("191999999999999999999", 10)
)

var (
	unitLPOTD = big.NewInt(262144)
)

var (
	unitInverse, _ = new(big.Int).SetString("78156646155174841979727994598816262306175212592076161876661508869554232690281", 10)
)

var (
	bigint0   = big.NewInt(0)
	bigint1   = big.NewInt(1)
	bigint2   = big.NewInt(2)
	bigint256 = big.NewInt(256)
)
