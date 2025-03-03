package scroll

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	l1GasPriceOracleAddress = "0x5300000000000000000000000000000000000002"
	methodL1BaseFee         = "l1BaseFee"
	methodL1CommitScalar    = "commitScalar"
	methodL1BlobScalar      = "blobScalar"
	methodL1BlobBaseFee     = "l1BlobBaseFee"
)

var (
	rlpDataLenOverhead = 1900
	rlpDataLenPerPool  = 370

	precision = bignumber.TenPowInt(9)
)
