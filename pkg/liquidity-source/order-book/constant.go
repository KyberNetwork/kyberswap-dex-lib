package orderbook

import (
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "orderbook"

	MaxAge = time.Minute
)

var (
	defaultGas = Gas{Base: 68331}
	gasByDex   = map[string]Gas{
		valueobject.ExchangePmm7: {Base: 922091},
	}
)
