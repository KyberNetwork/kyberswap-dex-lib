package entity

import "math/big"

type L2Fee struct {
	Decimals  *big.Int `json:"decimals"`
	L1BaseFee *big.Int `json:"l1BaseFee"`
	Overhead  *big.Int `json:"overhead"`
	Scalar    *big.Int `json:"scalar"`
}
