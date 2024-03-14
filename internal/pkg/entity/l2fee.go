package entity

import "math/big"

type (
	ScrollL1FeeParams struct {
		L1BaseFee *big.Int
		Overhead  *big.Int
		Scalar    *big.Int
	}
)
