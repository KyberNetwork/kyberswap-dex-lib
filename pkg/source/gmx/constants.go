package gmx

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexTypeGmx = "gmx"

	FlagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"

	defaultGas = 286524
)

var (
	BasisPointsDivisor = bignumber.BasisPoint
	PricePrecision     = bignumber.TenPowInt(30)
	USDGDecimals       = big.NewInt(18)
	OneUSD             = PricePrecision
)
