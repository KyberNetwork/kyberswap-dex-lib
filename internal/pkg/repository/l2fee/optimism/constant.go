package optimism

import (
	"math/big"

	"github.com/KyberNetwork/pool-service/pkg/util/bignumber"
)

const (
	gasPriceOracleAddress        = "0x420000000000000000000000000000000000000f"
	methodGetL1BaseFee           = "l1BaseFee"
	methodGetL1BlobBaseFee       = "blobBaseFee"
	methodGetL1BaseFeeScalar     = "baseFeeScalar"
	methodGetL1BlobBaseFeeScalar = "blobBaseFeeScalar"
)

var (
	fastLZDataLenOverhead = big.NewInt(990)
	fastLZDataLenPerPool  = big.NewInt(27)

	ecotonL1GasOverhead = big.NewInt(17600)
	ecotonL1GasPerPool  = big.NewInt(1000)

	oneMillion = bignumber.TenPowInt(6)
	sixteen    = big.NewInt(16)

	l1CostIntercept  = big.NewInt(-42585600)
	l1CostFastlzCoef = big.NewInt(836500)

	fjordDivisor   = bignumber.TenPowInt(12)
	ecotoneDivisor = big.NewInt(1000000 * 16)

	minTransactionSize       = big.NewInt(100)
	minTransactionSizeScaled = new(big.Int).Mul(minTransactionSize, oneMillion)
)
