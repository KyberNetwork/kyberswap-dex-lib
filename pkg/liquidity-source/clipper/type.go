package clipper

import "math/big"

type Extra struct {
	SwapsEnabled  bool
	K             float64
	TimeInSeconds int
	Assets        []PoolAsset
	Pairs         []PoolPair
}

type PoolAsset struct {
	Address       string
	Symbol        string
	Decimals      uint8
	PriceInUSD    float64
	Quantity      *big.Int
	ListingWeight int
}

type PoolPair struct {
	Assets           [2]string
	FeeInBasisPoints float64
}
