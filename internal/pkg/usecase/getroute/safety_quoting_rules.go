package getroute

import (
	"math/big"
	"strings"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
)

type SafetyQuoteCategory string

const (
	StrictlyStable SafetyQuoteCategory = "StrictlyStable"
	Stable         SafetyQuoteCategory = "Stable"
)

var (
	SafetyQuoteMappingDefault = map[SafetyQuoteCategory]float64{
		StrictlyStable: 0,
		Stable:         0.5,
	}
	// BasisPoint is one hundredth of 1 percentage point
	// https://en.wikipedia.org/wiki/Basis_point
	BasisPointMulByTen = big.NewInt(100000)
)

type SafetyQuoteReduction struct {
	deductionFactorInBps map[SafetyQuoteCategory]float64
	mu                   sync.RWMutex
	whiteListClients     mapset.Set[string]
}

func NewSafetyQuoteReduction(config valueobject.SafetyQuoteReductionConfig) *SafetyQuoteReduction {
	whitelistSet := whitelistClientToSet(config.WhitelistedClient)
	if len(config.Factor) == 0 {
		return &SafetyQuoteReduction{
			deductionFactorInBps: SafetyQuoteMappingDefault,
			whiteListClients:     whitelistSet,
		}
	}

	return &SafetyQuoteReduction{
		deductionFactorInBps: getFactor(config),
		whiteListClients:     whitelistSet,
	}
}

func whitelistClientToSet(clients []string) mapset.Set[string] {
	whitelistSet := mapset.NewSet[string]()
	for _, cli := range clients {
		whitelistSet.Add(strings.ToLower(cli))
	}

	return whitelistSet
}

func (f *SafetyQuoteReduction) GetSafetyQuotingRate(poolType string) float64 {
	switch poolType {
	case pooltypes.PoolTypes.LimitOrder, pooltypes.PoolTypes.KyberPMM,
		pooltypes.PoolTypes.HashflowV3, pooltypes.PoolTypes.NativeV1,
		pooltypes.PoolTypes.SwaapV2:
		return f.deductionFactorInBps[StrictlyStable]
	}

	return f.deductionFactorInBps[Stable]
}

// This function wrap the whole logic of safety quoting calculation
// which is describe in https://www.notion.so/kybernetwork/Safety-Quoting-for-KyberSwap-DEX-Aggregator-a673869729fe45adae8e1258ab6e43f4?pvs=4
func (f *SafetyQuoteReduction) Reduce(amount *pool.TokenAmount, deductionFactor float64, clientId string) pool.TokenAmount {
	if deductionFactor <= 0 || !f.whiteListClients.ContainsOne(strings.ToLower(clientId)) {
		return *amount
	}
	// convert deductionFactor from float to integer by multiply it by 10, then we will div (BasisPoint * 10)
	// 100% is equal to 10000Bps
	deductionFactorInBps := big.NewInt(int64(10 * (10000 - deductionFactor)))
	newAmount := new(big.Int).Div(
		new(big.Int).Mul(amount.Amount, deductionFactorInBps),
		BasisPointMulByTen,
	)

	return pool.TokenAmount{
		Token:  amount.Token,
		Amount: newAmount,
	}

}

func getFactor(config valueobject.SafetyQuoteReductionConfig) map[SafetyQuoteCategory]float64 {
	factors := map[SafetyQuoteCategory]float64{}
	for category, defaultVal := range SafetyQuoteMappingDefault {
		// only update safety quote reduction factor in SafetyQuoteMappingDefault
		// this protect SafetyQuoteReductionConfig from the wrong value in remote configs
		// if remote config doesn't contains enough value, default value will be used instead.
		if v, ok := config.Factor[string(category)]; !ok {
			factors[category] = defaultVal
		} else {
			factors[category] = v
		}

	}

	return factors
}

func compareFactor(x, y map[SafetyQuoteCategory]float64) bool {
	if len(x) != len(y) {
		return false
	}

	for k, xv := range x {
		if yv, ok := y[k]; !ok || !utils.Float64AlmostEqual(yv, xv) {
			return false
		}
	}

	return true
}

func (f *SafetyQuoteReduction) applyConfig(config valueobject.SafetyQuoteReductionConfig) {
	factor := getFactor(config)
	newClientList := whitelistClientToSet(config.WhitelistedClient)

	f.mu.Lock()
	defer f.mu.Unlock()
	// only apply cache only if it changed
	if !compareFactor(f.deductionFactorInBps, factor) {
		f.deductionFactorInBps = factor
	}
	if !newClientList.Equal(f.whiteListClients) {
		f.whiteListClients = newClientList
	}
}
