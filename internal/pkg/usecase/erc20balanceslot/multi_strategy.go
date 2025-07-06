package erc20balanceslot

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"

	repo "github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
)

type MultipleStrategyExtraParams struct {
	DoubleFromSource *DoubleFromSourceStrategyExtraParams
}

func (p *MultipleStrategyExtraParams) Dispatch(probe ProbeStrategy) ProbeStrategyExtraParams {
	switch probe.(type) {
	case *DoubleFromSourceStrategy:
		return p.DoubleFromSource
	}
	return nil
}

type MultipleStrategy struct {
	strategies []ProbeStrategy
}

func NewMultipleStrategy(rpcClient *rpc.Client, wallet common.Address) *MultipleStrategy {
	return &MultipleStrategy{
		strategies: []ProbeStrategy{
			NewWholeSlotStrategy(rpcClient, wallet),
			NewWholeSlotWithFStrategy(rpcClient, wallet),
			NewDoubleFromSourceStrategy(rpcClient),
		},
	}
}

func NewMultipleStrategyWithHoldersListAsFallback(rpcClient *rpc.Client, wallet common.Address, holdersListRepo *repo.HoldersListRedisRepositoryWithCache, watchlistRepo *repo.WatchlistRedisRepository) *MultipleStrategy {
	return &MultipleStrategy{
		strategies: []ProbeStrategy{
			NewWholeSlotStrategy(rpcClient, wallet),
			NewWholeSlotWithFStrategy(rpcClient, wallet),
			NewDoubleFromSourceStrategy(rpcClient),
			NewHoldersListStrategy(wallet, holdersListRepo, watchlistRepo),
		},
	}
}

func NewTestMultipleStrategy(strategies ...ProbeStrategy) *MultipleStrategy {
	return &MultipleStrategy{strategies: strategies}
}

func (p *MultipleStrategy) ProbeBalanceSlot(ctx context.Context, token common.Address, oldBalanceSlots *types.ERC20BalanceSlot, extraParams *MultipleStrategyExtraParams) (*types.ERC20BalanceSlot, error) {
	var (
		hadAttemptedList []string
		hadAttempted     = make(map[string]struct{})
		attempted        []string
		holdersListName  = (&HoldersListStrategy{}).Name(nil)
		bl               *types.ERC20BalanceSlot
		err              error
	)
	if oldBalanceSlots != nil {
		hadAttemptedList = oldBalanceSlots.StrategiesAttempted
		for _, strategy := range hadAttemptedList {
			hadAttempted[strategy] = struct{}{}
		}
	}
	for _, s := range p.strategies {
		_extraParams := extraParams.Dispatch(s)
		name := s.Name(_extraParams)
		if _, ok := hadAttempted[name]; ok {
			continue
		}
		// HoldersListStrategy can be attempted multiple times, so we don't record it
		if name != holdersListName {
			attempted = append(attempted, name)
		}
		bl, err = s.ProbeBalanceSlot(ctx, token, _extraParams)
		if err == nil {
			break
		}
		log.Ctx(ctx).Debug().Err(err).Msgf("strategy %s failed", name)
	}
	if len(attempted) == 0 {
		return nil, fmt.Errorf("there is no more strategies to attempted, already attempted %v", hadAttemptedList)
	}
	// found
	if bl != nil {
		bl.StrategiesAttempted = append(hadAttemptedList, attempted...)
		return bl, nil
	}
	// not found
	return &types.ERC20BalanceSlot{
		Token:               strings.ToLower(token.String()),
		Found:               false,
		StrategiesAttempted: append(hadAttemptedList, attempted...),
	}, nil
}
