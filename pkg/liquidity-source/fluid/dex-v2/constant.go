package dexv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/pkg/errors"
)

const (
	DexType         = "fluid-dex-v2"
	graphFirstLimit = 1000
)

var (
	defaultGas = Gas{BaseGas: 109334, CrossInitTickGas: 21492}

	ErrOverflow       = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier = errors.New("invalid feeTier")
	ErrTickNil        = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty   = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
)
