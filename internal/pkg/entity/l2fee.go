package entity

import "math/big"

type (
	ScrollL1FeeParams struct {
		L1BaseFee      *big.Int
		L1CommitScalar *big.Int
		L1BlobBaseFee  *big.Int
		L1BlobScalar   *big.Int
	}

	OptimismL1FeeParams struct {
		L1BaseFee           *big.Int
		L1BlobBaseFee       *big.Int
		L1BaseFeeScalar     *big.Int
		L1BlobBaseFeeScalar *big.Int
	}

	ArbitrumL1FeeParams struct {
		L1BaseFee *big.Int
	}
)
