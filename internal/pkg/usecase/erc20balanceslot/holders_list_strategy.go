package erc20balanceslot

import (
	"context"
	"errors"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/singleflight"

	repo "github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type HoldersListStrategy struct {
	wallet          common.Address
	holdersListRepo *repo.HoldersListRedisRepositoryWithCache
	watchlistRepo   *repo.WatchlistRedisRepository
	redisGroup      singleflight.Group
}

func NewHoldersListStrategy(wallet common.Address, holdersListRepo *repo.HoldersListRedisRepositoryWithCache, watchlistRepo *repo.WatchlistRedisRepository) *HoldersListStrategy {
	return &HoldersListStrategy{
		wallet:          wallet,
		holdersListRepo: holdersListRepo,
		watchlistRepo:   watchlistRepo,
	}
}

func (*HoldersListStrategy) Name(_ ProbeStrategyExtraParams) string {
	return "holders-list"
}

func (p *HoldersListStrategy) ProbeBalanceSlot(ctx context.Context, token common.Address, _ ProbeStrategyExtraParams) (*types.ERC20BalanceSlot, error) {
	logger.Debugf(ctx, "[%s] getting holders list for token %s", p.Name(nil), token)

	holdersList, err := p.holdersListRepo.Get(ctx, token)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// add to watchlist
			logger.WithFields(ctx, logger.Fields{"token": token}).Debugf("adding token to watchlist")
			_, err, _ := p.redisGroup.Do(strings.ToLower(token.String()), func() (interface{}, error) {
				err := p.watchlistRepo.Notify(ctx, token)
				return nil, err
			})
			if err != nil {
				logger.WithFields(ctx, logger.Fields{"token": token, "err": err}).Debugf("could not add token to watchlist")
			}
		}
		return nil, err
	}

	return &types.ERC20BalanceSlot{
		Token:   strings.ToLower(token.String()),
		Wallet:  strings.ToLower(p.wallet.String()),
		Found:   true,
		Holders: holdersList.Holders,
	}, nil
}
