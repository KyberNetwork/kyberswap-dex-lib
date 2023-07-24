package syncswapclassic

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	DefaultGas = syncswap.Gas{Swap: 300000}
	MaxFee     = bignumber.NewBig("100000")
)
