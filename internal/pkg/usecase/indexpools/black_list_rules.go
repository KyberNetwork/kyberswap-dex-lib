package indexpools

import (
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/samber/lo"
)

func (u *TradeDataGenerator) isWhitelistPoolHasReserves(p *entity.Pool) bool {
	if (len(p.Reserves)) == 0 {
		return false
	}

	// Pools that has less than 1 whitelist tokens which have positive reserve must be added to blacklist
	// some pools which have only 1 whitelist token that is non-zero reserve must be considered to be indexed
	// because it can result in good rate in some cases
	zeroReserveCount := 0
	totalWhitelistTokens := 0
	for i, token := range p.Tokens {
		if !u.config.WhitelistedTokenSet[strings.ToLower(token.Address)] {
			continue
		}
		totalWhitelistTokens++
		if !u.hasReserve(p.Reserves[i]) {
			zeroReserveCount += 1
		}
	}

	return totalWhitelistTokens-zeroReserveCount >= 1
}

func (u *TradeDataGenerator) removeZeroReservesPools(pools []*entity.Pool) ([]*entity.Pool, mapset.Set[string]) {
	zeroReserve := mapset.NewThreadUnsafeSet[string]()

	return lo.Filter(pools, func(p *entity.Pool, _ int) bool {
		hasReserve := u.isWhitelistPoolHasReserves(p)
		if !hasReserve {
			zeroReserve.Add(p.Address)
		}

		return hasReserve
	}), zeroReserve
}

func (u *TradeDataGenerator) hasReserve(reserve string) bool {
	if len(reserve) == 0 || reserve == "0" || reserve == "1" {
		return false
	}

	return true
}

func (u *TradeDataGenerator) getExhaustedReservesWhitelistPools(
	successed map[string]map[TradePair][]TradeData,
	failed map[string]map[TradePair][]TradeData) mapset.Set[string] {
	result := mapset.NewThreadUnsafeSet[string]()
	for p := range failed {
		if _, ok := successed[p]; ok {
			continue
		}

		result.Add(p)
	}

	return result
}
