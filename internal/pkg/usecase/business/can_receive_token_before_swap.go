package business

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	canReceiveTokenBeforeSwapExchange = map[valueobject.Exchange]bool{
		valueobject.ExchangeSushiSwap:       true,
		valueobject.ExchangeTrisolaris:      true,
		valueobject.ExchangeWannaSwap:       true,
		valueobject.ExchangeNearPad:         true,
		valueobject.ExchangePangolin:        true,
		valueobject.ExchangeTraderJoe:       true,
		valueobject.ExchangeLydia:           true,
		valueobject.ExchangeYetiSwap:        true,
		valueobject.ExchangeApeSwap:         true,
		valueobject.ExchangeJetSwap:         true,
		valueobject.ExchangeMDex:            true,
		valueobject.ExchangePancake:         true,
		valueobject.ExchangeWault:           true,
		valueobject.ExchangePancakeLegacy:   true,
		valueobject.ExchangeBiSwap:          true,
		valueobject.ExchangePantherSwap:     true,
		valueobject.ExchangeVVS:             true,
		valueobject.ExchangeCronaSwap:       true,
		valueobject.ExchangeCrodex:          true,
		valueobject.ExchangeMMF:             true,
		valueobject.ExchangeEmpireDex:       true,
		valueobject.ExchangePhotonSwap:      true,
		valueobject.ExchangeUniSwap:         true,
		valueobject.ExchangeShibaSwap:       true,
		valueobject.ExchangeDefiSwap:        true,
		valueobject.ExchangeSpookySwap:      true,
		valueobject.ExchangeSpiritSwap:      true,
		valueobject.ExchangePaintSwap:       true,
		valueobject.ExchangeMorpheus:        true,
		valueobject.ExchangeValleySwap:      true,
		valueobject.ExchangeYuzuSwap:        true,
		valueobject.ExchangeGemKeeper:       true,
		valueobject.ExchangeLizard:          true,
		valueobject.ExchangeValleySwapV2:    true,
		valueobject.ExchangeZipSwap:         true,
		valueobject.ExchangeQuickSwap:       true,
		valueobject.ExchangePolycat:         true,
		valueobject.ExchangeDFYN:            true,
		valueobject.ExchangePolyDex:         true,
		valueobject.ExchangeGravity:         true,
		valueobject.ExchangeCometh:          true,
		valueobject.ExchangeDinoSwap:        true,
		valueobject.ExchangeKrptoDex:        true,
		valueobject.ExchangeSafeSwap:        true,
		valueobject.ExchangeSwapr:           true,
		valueobject.ExchangeWagyuSwap:       true,
		valueobject.ExchangeAstroSwap:       true,
		valueobject.ExchangeDMM:             true,
		valueobject.ExchangeKyberSwap:       true,
		valueobject.ExchangeKyberSwapStatic: true,
		valueobject.ExchangeVelodrome:       true,
		valueobject.ExchangeDystopia:        true,
		valueobject.ExchangeChronos:         true,
		valueobject.ExchangeRamses:          true,
		valueobject.ExchangeVelocore:        true,
		valueobject.ExchangeCamelot:         true,

		// GMX and GMX-like exchanges are also able to receive token before calling swap.
		// However, they validate balance before swapping, so it's not possible to execute two gmx swaps consecutively without transferring token back to executor
		// I disable gmx exchanges here to reduce ad-hoc logic on back end side (do not allow two consecutive gmx swap)
		//valueobject.ExchangeGMX: true,
		//valueobject.ExchangeMadMex: true,
		//valueobject.ExchangeMetavault: true,
	}
)

func CanReceiveTokenBeforeSwap(exchange valueobject.Exchange) bool {
	return canReceiveTokenBeforeSwapExchange[exchange]
}
