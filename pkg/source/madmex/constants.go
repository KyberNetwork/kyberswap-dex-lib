package madmex

import "math/big"

const DexTypeMadmex = "madmex"

const flagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"

var secondaryPriceFeedVersionByChainID = map[ChainID]SecondaryPriceFeedVersion{
	POLYGON: SecondaryPriceFeedVersion1,
}

var (
	DefaultGas             = Gas{Swap: 165000}
	BasisPointsDivisor     = big.NewInt(10000)
	PricePrecision         = new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	USDGDecimals           = big.NewInt(18)
	OneUSD                 = PricePrecision
	FlagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"
)
