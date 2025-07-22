package nomiswapstable

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nomiswap"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	defaultGas  = nomiswap.Gas{Swap: 300000}
	A_PRECISION = u256.U100
	MaxFee      = uint256.NewInt(100000)
)

const (
	MAX_LOOP_LIMIT = 256
)
