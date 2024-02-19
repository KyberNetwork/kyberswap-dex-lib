package erc20balanceslot

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	repo "github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/pkg/logger"
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

func (p *MultipleStrategy) ProbeBalanceSlot(ctx context.Context, token common.Address, oldBalanceSlots *entity.ERC20BalanceSlot, extraParams *MultipleStrategyExtraParams) (*entity.ERC20BalanceSlot, error) {
	var (
		hadAttemptedList []string
		hadAttempted     = make(map[string]struct{})
		attempted        []string
		holdersListName  = (&HoldersListStrategy{}).Name(nil)
		bl               *entity.ERC20BalanceSlot
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
		logger.Debugf(ctx, "strategy %s failed: %s", name, err)
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
	return &entity.ERC20BalanceSlot{
		Token:               strings.ToLower(token.String()),
		Found:               false,
		StrategiesAttempted: append(hadAttemptedList, attempted...),
	}, nil
}
