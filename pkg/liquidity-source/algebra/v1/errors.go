package algebrav1

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrUnmarshalVolLiq     = errors.New("failed to unmarshal volumePerLiquidityInBlock")
	ErrMaxBinarySearchLoop = errors.New("max binary search loop reached")
	ErrStaleTimepoints     = errors.New("getting stale timepoint data")
	ErrTickNil             = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrTickInvalid         = errors.New("tick is invalid")
	ErrTicksEmpty          = errors.New("ticks list is empty")
	ErrInvalidToken        = errors.New("invalid token")
	ErrZeroAmountIn        = errors.New("amountIn is 0")
	ErrZeroAmountOut       = errors.New("amountOut is 0")
	ErrSPL                 = errors.New("invalid sqrt price limit")
	ErrPoolLocked          = errors.New("pool is locked")
	ErrOverflow            = errors.New("bigInt overflow int/uint256")

	ErrNotSupportFetchFullTick = errors.New("not support fetching full ticks")
)
