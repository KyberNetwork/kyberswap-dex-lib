package aave

import (
	"time"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
)

var (
	DefaultGas = curve.Gas{Exchange: 495000}
	Precision  = uint256.MustFromDecimal("1000000000000000000")

	DepositFrozen = time.Now().Unix() > 1750939722 // https://vote.onaave.com/proposal/?proposalId=333
)
