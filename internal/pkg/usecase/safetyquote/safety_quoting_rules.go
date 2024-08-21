package safetyquote

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
)

type SafetyQuoteReduction struct {
	// These configs are not refreshed, instead the whole object is renew
	excludeOneSwapEnable bool
	deductionFactorInBps map[types.SafetyQuoteCategory]float64
	whiteListClients     mapset.Set[string]
	tokenGroups          *valueobject.TokenGroupConfig
}

func NewSafetyQuoteReduction(config *valueobject.SafetyQuoteReductionConfig) *SafetyQuoteReduction {
	whitelistSet := whitelistClientToSet(config.WhitelistedClient)

	if len(config.Factor) == 0 {
		return &SafetyQuoteReduction{
			excludeOneSwapEnable: true,
			deductionFactorInBps: types.SafetyQuoteMappingDefault,
			whiteListClients:     whitelistSet,
			tokenGroups:          config.TokenGroupConfig,
		}
	}

	return &SafetyQuoteReduction{
		excludeOneSwapEnable: config.ExcludeOneSwapEnable,
		deductionFactorInBps: getFactor(config),
		whiteListClients:     whitelistSet,
		tokenGroups:          config.TokenGroupConfig,
	}
}

func whitelistClientToSet(clients []string) mapset.Set[string] {
	whitelistSet := mapset.NewThreadUnsafeSet[string]()
	for _, cli := range clients {
		whitelistSet.Add(strings.ToLower(cli))
	}

	return whitelistSet
}

func (f *SafetyQuoteReduction) GetSafetyQuotingRate(params types.SafetyQuotingParams) float64 {
	if f.whiteListClients.ContainsOne(strings.ToLower(params.ClientId)) {
		return 0
	}
	if f.excludeOneSwapEnable && params.ApplyDeductionFactor {
		return 0
	}

	// Check safety quoting rate by pool types
	switch params.PoolType {
	case pooltypes.PoolTypes.LimitOrder, pooltypes.PoolTypes.KyberPMM,
		pooltypes.PoolTypes.HashflowV3, pooltypes.PoolTypes.NativeV1,
		pooltypes.PoolTypes.SwaapV2:
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

// This function wrap the whole logic of safety quoting calculation
// which is describe in https://www.notion.so/kybernetwork/Safety-Quoting-for-KyberSwap-DEX-Aggregator-a673869729fe45adae8e1258ab6e43f4?pvs=4
func (f *SafetyQuoteReduction) Reduce(amount *pool.TokenAmount, deductionFactor float64) pool.TokenAmount {
	if deductionFactor <= 0 {
		return *amount
	}
	// convert deductionFactor from float to integer by multiply it by 10, then we will div (BasisPoint * 10)
	// 100% is equal to 10000Bps
	deductionFactorInBps := big.NewInt(int64(10 * (10000 - deductionFactor)))
	newAmount := new(big.Int).Div(
		new(big.Int).Mul(amount.Amount, deductionFactorInBps),
		types.BasisPointMulByTen,
	)

	return pool.TokenAmount{
		Token:  amount.Token,
		Amount: newAmount,
	}

}

func getFactor(config *valueobject.SafetyQuoteReductionConfig) map[types.SafetyQuoteCategory]float64 {
	factors := map[types.SafetyQuoteCategory]float64{}
	for category, defaultVal := range types.SafetyQuoteMappingDefault {
		// only update safety quote reduction factor in SafetyQuoteMappingDefault
		// this protect SafetyQuoteReductionConfig from the wrong value in remote configs
		// if remote config doesn't contains enough value, default value will be used instead.
		if v, ok := config.Factor[strings.ToLower(string(category))]; !ok {
			factors[category] = defaultVal
		} else {
			factors[category] = v
		}

	}

	return factors
}
