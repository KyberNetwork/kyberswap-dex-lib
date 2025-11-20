package stable

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

const DexType = "lista-stable"

var (
	Precision      = bignumber.NewBig10("1000000000000000000") // 1e18
	FeeDenominator = bignumber.NewBig10("10000000000")         // 1e10
)
