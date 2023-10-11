package erc20balanceslot

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type MultipleStrategyExtraParams struct {
}

func (p *MultipleStrategyExtraParams) Dispatch(probe ProbeStrategy) ProbeStrategyExtraParams {
	return nil
}

type MultipleStrategy struct {
	strategies []ProbeStrategy
}

func NewMultipleStrategy(rpcClient *rpc.Client, wallet common.Address) *MultipleStrategy {
	return &MultipleStrategy{
		strategies: []ProbeStrategy{
			NewWholeSlotStrategy(rpcClient, wallet),
		},
	}
}

func NewTestMultipleStrategy(strategies ...ProbeStrategy) *MultipleStrategy {
	return &MultipleStrategy{strategies: strategies}
}

func (p *MultipleStrategy) ProbeBalanceSlot(token common.Address, oldBalanceSlots *entity.ERC20BalanceSlot, extraParams *MultipleStrategyExtraParams) (*entity.ERC20BalanceSlot, error) {
	var (
		hadAttemptedList []string
		hadAttempted     = make(map[string]struct{})
		attempted        []string
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
		if _, ok := hadAttempted[s.Name()]; ok {
			continue
		}
		attempted = append(attempted, s.Name())
		bl, err = s.ProbeBalanceSlot(token, extraParams.Dispatch(s))
		if err == nil {
			break
		}
		logger.Debugf("strategy %s failed: %s", s.Name(), err)
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
