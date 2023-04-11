package valueobject

import "math/big"

type GasOption struct {
	GasFeeInclude bool
	Price         *big.Float
	TokenPrice    float64
}
