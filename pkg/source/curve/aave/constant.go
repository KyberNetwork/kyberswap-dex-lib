package aave

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	DefaultGas = curve.Gas{Exchange: 495000}
	Precision  = bignumber.NewBig10("1000000000000000000")
)
