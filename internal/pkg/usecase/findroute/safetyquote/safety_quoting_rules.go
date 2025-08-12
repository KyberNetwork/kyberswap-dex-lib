package safetyquote

import (
	"math"
	"math/big"
	"strings"
	"time"

	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/samber/lo"
	"github.com/zeebo/xxh3"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Reduction struct {
	// These configs are not refreshed, instead the whole object is renewed
	excludeOneSwapEnable bool
	deductionFactorInBps map[types.SafetyQuoteCategory]float64
	randomJitter         float64
	whitelistedClients   mapset.Set[string]
	tokenGroups          valueobject.TokenGroupConfig
}

func NewSafetyQuoteReduction(config *valueobject.SafetyQuoteReductionConfig) *Reduction {
	whitelistedClients := toLowerStrSet(config.WhitelistedClient)
	tokenGroups := lo.FromPtr(config.TokenGroupConfig)

	if len(config.Factor) == 0 {
		return &Reduction{
			excludeOneSwapEnable: true,
			deductionFactorInBps: types.SafetyQuoteMappingDefault,
			whitelistedClients:   whitelistedClients,
			tokenGroups:          tokenGroups,
		}
	}

	return &Reduction{
		excludeOneSwapEnable: config.ExcludeOneSwapEnable,
		deductionFactorInBps: getFactor(config),
		randomJitter:         config.RandomJitter,
		whitelistedClients:   whitelistedClients,
		tokenGroups:          tokenGroups,
	}
}

func toLowerStrSet(strs []string) mapset.Set[string] {
	set := mapset.NewThreadUnsafeSetWithSize[string](len(strs))
	for _, str := range strs {
		set.Add(strings.ToLower(str))
	}
	return set
}

// Reduce wraps the whole logic of safety quoting calculation
// which is described in https://www.notion.so/kybernetwork/Safety-Quoting-for-KyberSwap-DEX-Aggregator-a673869729fe45adae8e1258ab6e43f4?pvs=4
// Deduction factor can be positive in optimistic case
func (f *Reduction) Reduce(params types.SafetyQuotingParams) *big.Int {
	deductionFactor := f.getSafetyQuotingRate(params)
	if deductionFactor == 0 {
		return params.Amount
	} else if f.randomJitter > 0 {
		deductionFactor += deductionFactor * f.randomJitter * (2*f.rand(params.Address) - 1)
	}

	amountF, _ := params.Amount.Float64()
	amountF *= (10000 - deductionFactor) / 10000
	newAmount, _ := big.NewFloat(amountF).Int(nil)

	return newAmount
}

func (f *Reduction) getSafetyQuotingRate(params types.SafetyQuotingParams) float64 {
	if f.whitelistedClients.ContainsOne(strings.ToLower(params.ClientId)) {
		return 0
	} else if f.excludeOneSwapEnable && params.HasOnlyOneSwap {
		return 0
	}

	// Check converter exchanges
	switch params.Exchange {
	case dexValueObject.ExchangeFrxETH, dexValueObject.ExchangeDaiUsds,
		dexValueObject.ExchangeUsd0PP, dexValueObject.ExchangeOETH,
		dexValueObject.ExchangePolMatic, dexValueObject.ExchangeEtherFieBTC,
		dexValueObject.ExchangeHoney, dexValueObject.ExchangeUsdsLitePsm,
		dexValueObject.ExchangeERC4626:
		return f.deductionFactorInBps[types.Converter]
	}

	// Check safety quoting rate by pool types
	if dexValueObject.IsRFQSource(params.Exchange) {
		return f.deductionFactorInBps[types.StrictlyStable]
	}

	// Check safety quoting rate by tokens
	// Reference: https://www.notion.so/kybernetwork/Stable-and-Correlated-Tokens-data-d1bdc7ad1ec14d8ebeab031c493e730e
	if f.tokenGroups.StableGroup[params.TokenIn] && f.tokenGroups.StableGroup[params.TokenOut] {
		return f.deductionFactorInBps[types.Stable]
	} else if f.tokenGroups.CorrelatedGroup1[params.TokenIn] && f.tokenGroups.CorrelatedGroup1[params.TokenOut] {
		return f.deductionFactorInBps[types.Correlated]
	} else if f.tokenGroups.CorrelatedGroup2[params.TokenIn] && f.tokenGroups.CorrelatedGroup2[params.TokenOut] {
		return f.deductionFactorInBps[types.Correlated]
	} else if f.tokenGroups.CorrelatedGroup3[params.TokenIn] && f.tokenGroups.CorrelatedGroup3[params.TokenOut] {
		return f.deductionFactorInBps[types.Correlated]
	}

	return f.deductionFactorInBps[types.Default]
}

// rand generates a non-cryptographic random number in [0,1] that stays the same for 6 seconds for the same seed
func (f *Reduction) rand(seed string) float64 {
	h := xxh3.HashStringSeed(seed, uint64(time.Now().Unix()/6))
	return float64(h) / math.MaxUint64
}

// Because viper makes all keys case-insensitive, so that we have to accept case-insensitive in safety quoting configs
// which receives values from both source viper config and ks-settings
// Ref: https://github.com/spf13/viper#does-viper-support-case-sensitive-keys
func getFactor(config *valueobject.SafetyQuoteReductionConfig) map[types.SafetyQuoteCategory]float64 {
	factors := map[types.SafetyQuoteCategory]float64{}
	for category, defaultVal := range types.SafetyQuoteMappingDefault {
		// only update safety quote reduction factor in SafetyQuoteMappingDefault
		// this protect SafetyQuoteReductionConfig from the wrong value in remote configs
		// if remote config doesn't contain enough value, default value will be used instead.
		if v, ok := config.Factor[string(category)]; ok {
			factors[category] = v
		} else if value, ok := config.Factor[strings.ToLower(string(category))]; ok {
			factors[category] = value
		} else {
			factors[category] = defaultVal
		}

	}

	return factors
}
