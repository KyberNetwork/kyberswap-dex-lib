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
		valueobject.ExchangePmm7:      {Base: 922091},
		valueobject.ExchangePmm13:     {Base: 574269},
		"pmm-20":                      {Base: 700000}, // Rave: adapter fill of an Ondo token, 673-689k measured on bsc
		valueobject.ExchangeNativeV2:  {Base: 144648},
		valueobject.ExchangePrismProp: {Base: 207608},
	}
)
