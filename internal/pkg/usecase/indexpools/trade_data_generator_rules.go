package indexpools

import (
	"errors"

	defaultpmm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/default-pmm"
	mxtrading "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/mx-trading"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
	dexT1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-t1"
	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/integral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch"
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v1"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func (u *TradeDataGenerator) removeZeroReservesPools(pools []*entity.Pool) ([]*entity.Pool, mapset.Set[string]) {
	zeroReserve := mapset.NewThreadUnsafeSet[string]()

	return lo.Filter(pools, func(p *entity.Pool, _ int) bool {
		hasReserve := p.HasReserves()
		if !hasReserve {
			zeroReserve.Add(p.Address)
		}

		return hasReserve
	}), zeroReserve
}

func (u *TradeDataGenerator) hasReserve(poolSim poolpkg.IPoolSimulator, pool *entity.Pool, token string) bool {
	// in curve family, index might be greater than reserves len
	i := poolSim.GetTokenIndex(token)
	if i < 0 || i >= len(poolSim.GetReserves()) {
		return true
	}
	reserve := pool.Reserves[i]
	if len(reserve) == 0 || reserve == "0" || reserve == "1" {
		return false
	}

	return true
}

func (u *TradeDataGenerator) filterFailedPools(
	successed map[TradeDataId]*LiquidityScoreCalcInput,
	failed map[TradeDataId]*LiquidityScoreCalcInput) mapset.Set[string] {
	result := mapset.NewThreadUnsafeSet[string]()
	for tradeId := range failed {
		if _, ok := successed[tradeId]; ok {
			continue
		}

		result.Add(tradeId.Pool)
	}

	return result
}

func (u *TradeDataGenerator) getPairType(tokenI, tokenJ string) []valueobject.TradeDataType {
	result := make([]valueobject.TradeDataType, 0, 4)
	isTokenIWhitelist := u.config.WhitelistedTokenSet[tokenI]
	isTokenJWhitelist := u.config.WhitelistedTokenSet[tokenJ]

	if isTokenIWhitelist && isTokenJWhitelist {
		result = append(result, valueobject.WHITELIST_WHITELIST)
	}

	if isTokenIWhitelist {
		result = append(result, valueobject.TOKEN_WHITELIST)
	}

	if isTokenJWhitelist {
		result = append(result, valueobject.WHITELIST_TOKEN)
	}

	result = append(result, valueobject.DIRECT)

	return result
}

func (gen *TradeDataGenerator) poolHasLimitCheck(exchange string) bool {
	if exchange == dexValueObject.ExchangeKyberSwapLimitOrder || exchange == dexValueObject.ExchangeKyberSwapLimitOrderDS {
		return false
	}
	if exchange == dexValueObject.ExchangeIntegral ||
		exchange == dexValueObject.ExchangeFluidDexT1 {
		return true
	}

	return dexValueObject.IsRFQSource(valueobject.Exchange(exchange))
}

func (gen *TradeDataGenerator) errAmountInLessThanMinAllowed(dex dexValueObject.Exchange, err error) bool {
	if err == nil {
		return false
	}
	switch dex {
	case dexValueObject.ExchangeHashflowV3:
		return errors.Is(err, hashflowv3.ErrAmtInLessThanMinAllowed)
	case dexValueObject.ExchangeDexalot:
		return errors.Is(err, dexalot.ErrAmountInIsLessThanLowestPriceLevel)
	case dexValueObject.ExchangeClipper:
		return errors.Is(err, clipper.ErrMinAmountInNotEnough)
	case dexValueObject.ExchangePmm2, dexValueObject.ExchangePmm3:
		return errors.Is(err, defaultpmm.ErrAmountInIsLessThanLowestPriceLevel)
	case dexValueObject.ExchangePmm1:
		return errors.Is(err, mxtrading.ErrAmountInIsLessThanLowestPriceLevel)
	case dexValueObject.ExchangeNativeV1:
		return errors.Is(err, nativev1.ErrAmountInIsLessThanLowestPriceLevel)
	case dexValueObject.ExchangeIntegral:
		return errors.Is(err, integral.ErrTR03)
	case dexValueObject.ExchangeFluidDexT1:
		return errors.Is(err, dexT1.ErrInvalidAmountIn)
	case dexValueObject.ExchangeLO1inch:
		return errors.Is(err, lo1inch.ErrCannotFulfillAmountIn)
	}

	return false
}
