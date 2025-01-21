package gmx

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const DexTypeGmx = "gmx"

const FlagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"

var (
	DefaultGas         = Gas{Swap: 165000}
	BasisPointsDivisor = bignumber.BasisPoint
	PricePrecision     = bignumber.TenPowInt(30)
	USDGDecimals       = big.NewInt(18)
	OneUSD             = PricePrecision
)
