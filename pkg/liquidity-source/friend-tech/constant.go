package friendtech

import (
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const (
	DexType = "friend-tech"
)

var (
	PoolAddress = "0x7cfc830448484cdf830625373820241e61ef4acf"
	Tokens      = []*entity.PoolToken{
		{
			Address:   strings.ToLower("0x0bd4887f7d41b35cd75dff9ffee2856106f86670"),
			Symbol:    "FRIEND",
			Decimals:  18,
			Name:      "Friend",
			Swappable: true,
		},
		{
			Address:   strings.ToLower("0x4200000000000000000000000000000000000006"),
			Symbol:    "WETH",
			Decimals:  18,
			Name:      "Wrapped Ether",
			Swappable: true,
		},
	}
)

var (
	defaultGas = Gas{Swap: 60000}
)

const (
	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"

	meerkatPairMethodSwapFee                   = "swapFee"
	mdexFactoryMethodGetPairFees               = "getPairFees"
	shibaswapPairMethodTotalFee                = "totalFee"
	croDefiSwapFactoryMethodTotalFeeBasisPoint = "totalFeeBasisPoint"
	zkSwapFinancePairMethodGetSwapFee          = "getSwapFee"
)

const (
	FeeTrackerIDMMF         = "mmf"
	FeeTrackerIDMdex        = "mdex"
	FeeTrackerIDShibaswap   = "shibaswap"
	FeeTrackerIDDefiswap    = "defiswap"
	FeeTrackerZKSwapFinance = "zkswap-finance"
)
