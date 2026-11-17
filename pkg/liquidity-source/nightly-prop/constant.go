package nightlyprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNightlyProp

	routerMethodGetRoutes = "getRoutes"
	poolMethodGetTokens   = "getTokens"
)

var ErrInvalidRouterAddress = errors.New("nightly-prop: invalid router address")
