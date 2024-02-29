package valueobject

import (
	"hash/fnv"
	"sort"

	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Exchange = dexValueObject.Exchange

func IsAnExchange(exchange Exchange) bool {
	var contained bool
	_, contained = dexValueObject.AMMSourceSet[exchange]
	if contained {
		return true
	}

	_, contained = RFQSourceSet[exchange]
	return contained
}

func GetSourcesAsSlice(sources map[Exchange]struct{}) []string {
	result := make([]string, len(sources))
	count := 0
	for src := range sources {
		result[count] = string(src)
		count = count + 1
	}
	return result
}

var RFQSourceSet = map[Exchange]struct{}{
	dexValueObject.ExchangeKyberPMM: {},

	dexValueObject.ExchangeKyberSwapLimitOrderDS: {},

	dexValueObject.ExchangeSwaapV2: {},
}

func IsRFQSource(exchange Exchange) bool {
	_, contained := RFQSourceSet[exchange]

	return contained
}

// HashSources unique, then sort and has the slice string
func HashSources(sources []string) uint64 {
	// Step 1: Make the elements unique
	uniqueMap := make(map[string]bool)
	for _, str := range sources {
		uniqueMap[str] = true
	}

	// Extract the unique elements
	uniqueSlice := make([]string, 0, len(uniqueMap))
	for str := range uniqueMap {
		uniqueSlice = append(uniqueSlice, str)
	}

	// Step 2: Sort the unique elements stably
	sort.Strings(uniqueSlice)

	// Step 3: Hash
	h := fnv.New64()
	for _, str := range uniqueSlice {
		_, _ = h.Write([]byte(str))
	}
	return h.Sum64()
}

// `useApproveMaxExchangeSet` defines set of exchanges that we should track if executor `approveMax` for the pools,
// to support optimizing gas cost when encode (not all pools need to call `approveMax` when swap with executor).
// This data can be add by check the SC code, if the function of each pool type reads the SHOULD_APPROVE_MAX flag.
var useApproveMaxExchangeSet = map[Exchange]struct{}{
	// executeStableSwap
	dexValueObject.ExchangeOneSwap:             {},
	dexValueObject.ExchangeNerve:               {},
	dexValueObject.ExchangeIronStable:          {},
	dexValueObject.ExchangeSynapse:             {},
	dexValueObject.ExchangeSaddle:              {},
	dexValueObject.ExchangeAxial:               {},
	dexValueObject.ExchangeAlienBaseStableSwap: {},

	// executeCurve
	dexValueObject.ExchangeCurve:         {},
	dexValueObject.ExchangeEllipsis:      {},
	dexValueObject.ExchangeKokonutCrypto: {},

	dexValueObject.ExchangeCurveStablePlain: {},

	// executePancakeStableSwap
	dexValueObject.ExchangePancakeStable: {},

	// executeBalV2
	dexValueObject.ExchangeBalancerV2Weighted:         {},
	dexValueObject.ExchangeBalancerV2Stable:           {},
	dexValueObject.ExchangeBalancerV2ComposableStable: {},
	dexValueObject.ExchangeBeethovenXWeighted:         {},
	dexValueObject.ExchangeBeethovenXStable:           {},
	dexValueObject.ExchangeBeethovenXComposableStable: {},
	dexValueObject.ExchangeGyroscope2CLP:              {},
	dexValueObject.ExchangeGyroscope3CLP:              {},
	dexValueObject.ExchangeGyroscopeECLP:              {},

	// executeBalancerV1
	dexValueObject.ExchangeBalancerV1: {},

	// executeDODO
	dexValueObject.ExchangeDodo: {},

	// executeHashflow

	// executeWrappedstETH
	dexValueObject.ExchangeMakerLido: {},

	// executePlatypus
	dexValueObject.ExchangePlatypus: {},

	// executePSM
	dexValueObject.ExchangeMakerPSM: {},

	// executeBalancerBatch

	// executeMantis
	dexValueObject.ExchangeMantisSwap: {},

	// executeWombat
	dexValueObject.ExchangeWombat: {},

	// executeRfq
	dexValueObject.ExchangeKyberPMM: {},

	// executeKyberDSLO
	dexValueObject.ExchangeKyberSwapLimitOrderDS: {},

	// executeVooi
	dexValueObject.ExchangeVooi: {},

	// executeMaticMigrate
	dexValueObject.ExchangePolMatic: {},

	// executeSmardex
	dexValueObject.ExchangeSmardex: {},

	// executeVelocoreV2
	dexValueObject.ExchangeVelocoreV2CPMM:         {},
	dexValueObject.ExchangeVelocoreV2WombatStable: {},

	// executeSwaapV2
	dexValueObject.ExchangeSwaapV2: {},

	// executeBancorV3
	dexValueObject.ExchangeBancorV3: {},

	// executeEtherFiWeETH
	dexValueObject.ExchangeEtherfiWEETH: {},

	// executeKelp
	dexValueObject.ExchangeKelpRSETH: {},
}

func IsApproveMaxExchange(exchange Exchange) bool {
	_, ok := useApproveMaxExchangeSet[exchange]
	return ok
}
