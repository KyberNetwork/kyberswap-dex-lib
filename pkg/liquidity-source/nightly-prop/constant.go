package nightlyprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNightlyProp

	routerMethodGetRoutes      = "getRoutes"
	routerMethodGetPoolStateV1 = "getPoolStateV1"
	poolMethodGetTokens        = "getTokens"
)

var (
	ErrInvalidRouterAddress = errors.New("nightly-prop: invalid router address")
	ErrInvalidPoolRecord    = errors.New("nightly-prop: invalid pool record")
)
