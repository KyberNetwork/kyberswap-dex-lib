package valueobject

import (
	"hash/fnv"
	"sort"

	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Exchange = dexValueObject.Exchange

func IsAnExchange(exchange Exchange) bool {
	return dexValueObject.IsAMMSource(exchange) || dexValueObject.IsRFQSource(exchange)
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

	dexValueObject.ExchangeCurveStablePlain:  {},
	dexValueObject.ExchangeCurveStableNg:     {},
	dexValueObject.ExchangeCurveStableMetaNg: {},
	dexValueObject.ExchangeCurveTriCryptoNg:  {},
	dexValueObject.ExchangeCurveTwoCryptoNg:  {},

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
	dexValueObject.ExchangeDodo:               {},
	dexValueObject.ExchangeDodoClassical:      {},
	dexValueObject.ExchangeDodoPrivatePool:    {},
	dexValueObject.ExchangeDodoStablePool:     {},
	dexValueObject.ExchangeDodoVendingMachine: {},

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

	// executeIntegral
	dexValueObject.ExchangeIntegral: {},

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

	// executeRocketPool
	dexValueObject.ExchangeRocketPoolRETH: {},

	// executeEthenaSusde
	dexValueObject.ExchangeEthenaSusde: {},

	// executeMakersDAI
	dexValueObject.ExchangeMakerSavingsDai: {},

	// executeHashflow
	dexValueObject.ExchangeHashflowV3: {},

	// executeNative
	dexValueObject.ExchangeNativeV1: {},

	// executeRenzo
	dexValueObject.ExchangeRenzoEZETH: {},

	// executePufferFinance
	dexValueObject.ExchangePufferPufETH: {},

	// executeAmbient
	dexValueObject.ExchangeAmbient: {},

	// executeEtherVista
	dexValueObject.ExchangeEtherVista: {},

	// executeLitePSM
	dexValueObject.ExchangeLitePSM: {},

	// executeMkrSky
	dexValueObject.ExchangeMkrSky: {},

	// executeDaiUsds
	dexValueObject.ExchangeDaiUsds: {},

	// executeFluidVaultT1
	dexValueObject.ExchangeFluidVaultT1: {},

	// executeFluidDexT1
	dexValueObject.ExchangeFluidDexT1: {},

	// executeUsd0PP
	dexValueObject.ExchangeUsd0PP: {},

	// executeRingSwap
	dexValueObject.ExchangeRingSwap: {},

	// executeBebop
	dexValueObject.ExchangeBebop: {},

	// executeWBETH
	dexValueObject.ExchangeWBETH: {},

	// executeUniswapV1
	dexValueObject.ExchangeUniSwapV1: {},

	// executeMantleUSD
	dexValueObject.ExchangeOndoUSDY: {},

	// executeDexalot
	dexValueObject.ExchangeDexalot: {},

	// executeSfrxETHConvertor
	dexValueObject.ExchangeSfrxETHConvertor: {},

	// executeEETHOrWeETH
	dexValueObject.ExchangeEtherfiVampire: {},

	// executeOneInchV6Rfq
	dexValueObject.ExchangeMxTrading: {},
}

func IsApproveMaxExchange(exchange Exchange) bool {
	_, ok := useApproveMaxExchangeSet[exchange]
	return ok
}
