package erc20balanceslot

import (
	"context"
	"errors"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"

	repo "github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
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
	log.Ctx(ctx).Debug().Msgf("[%s] getting holders list for token %s", p.Name(nil), token)

	holdersList, err := p.holdersListRepo.Get(ctx, token)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// add to watchlist
			log.Ctx(ctx).Debug().Stringer("token", token).Msg("adding token to watchlist")
			_, err, _ := p.redisGroup.Do(strings.ToLower(token.String()), func() (interface{}, error) {
				err := p.watchlistRepo.Notify(ctx, token)
				return nil, err
			})
			if err != nil {
				log.Ctx(ctx).Debug().Err(err).Stringer("token", token).Msg("could not add token to watchlist")
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
