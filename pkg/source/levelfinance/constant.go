package levelfinance

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

var (
	DefaultGas = Gas{Swap: 125000}

	Precision = bignumber.TenPowInt(10)
)
