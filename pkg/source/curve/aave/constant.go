package aave

import (
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	DefaultGas = curve.Gas{Exchange: 495000}
	Precision  = bignumber.NewBig10("1000000000000000000")

	DepositFrozen = time.Now().Unix() > 1750939722 // https://vote.onaave.com/proposal/?proposalId=333
)
