package generic_simple_rate

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "generic-simple-rate"

	defaultReserves       = "100000000000000000000000000"
	DefaultGas      int64 = 60000
)

var (
	ErrPoolPaused = errors.New("pool is paused")
	ErrOverflow   = errors.New("overflow")

	supportNativeSwapExchanges = map[string]struct{}{
		valueobject.ExchangeFrxETH: {},
		valueobject.ExchangeOETH:   {},
		valueobject.ExchangeWBETH:  {},
	}
)
