package limitorder

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

const prefixLimitOrderPoolID = "limit_order_pool"

func (l *limitOrder) initPool(ctx context.Context, pair *valueobject.TokenPair) (entity.Pool, error) {
	newPool := entity.Pool{
		Address:  l.getPoolID(pair.Token0, pair.Token1),
		Exchange: l.scanDexCfg.Id,
		Type:     constant.PoolTypes.LimitOrder,
	}
	token0, err := l.scanService.FetchOrGetToken(ctx, pair.Token0)
	if err != nil {
		logger.Warnf("cannot get tokenInfo for Token0=%s and  cause by %v", pair.Token0, err)
		return newPool, err
	}
	token1, err := l.scanService.FetchOrGetToken(ctx, pair.Token1)
	if err != nil {
		logger.Warnf("cannot get tokenInfo for Token1=%s and  cause by %v", pair.Token1, err)
		return newPool, err
	}
	if strings.ToLower(token0.Address) > strings.ToLower(token1.Address) {
		newPool.Tokens = []*entity.PoolToken{
			{
				Address:   token0.Address,
				Name:      token0.Name,
				Symbol:    token0.Symbol,
				Decimals:  token0.Decimals,
				Swappable: true,
			}, {
				Address:   token1.Address,
				Name:      token1.Name,
				Symbol:    token1.Symbol,
				Decimals:  token1.Decimals,
				Swappable: true,
			}}
	} else {
		newPool.Tokens = []*entity.PoolToken{
			{
				Address:   token1.Address,
				Name:      token1.Name,
				Symbol:    token1.Symbol,
				Decimals:  token1.Decimals,
				Swappable: true,
			},
			{
				Address:   token0.Address,
				Name:      token0.Name,
				Symbol:    token0.Symbol,
				Decimals:  token0.Decimals,
				Swappable: true,
			},
		}
	}
	return newPool, nil
}

func (l *limitOrder) getPoolID(token0, token1 string) string {
	lowerToken0, lowerToken1 := strings.ToLower(token0), strings.ToLower(token1)
	if lowerToken0 > lowerToken1 {
		return fmt.Sprintf("%s_%s_%s", prefixLimitOrderPoolID, lowerToken0, lowerToken1)
	}
	return fmt.Sprintf("%s_%s_%s", prefixLimitOrderPoolID, lowerToken1, lowerToken0)
}
