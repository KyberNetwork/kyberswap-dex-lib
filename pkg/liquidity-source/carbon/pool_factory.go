package carbon

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryCE(DexType, NewPoolFactory)

type PoolFactory struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	controller   common.Address
}

func NewPoolFactory(config *Config, ethrpcClient *ethrpc.Client) *PoolFactory {
	return &PoolFactory{
		cfg:          config,
		ethrpcClient: ethrpcClient,
		controller:   config.Controller,
	}
}

func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	return nil, nil
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	return true
}

func (f *PoolFactory) DecodePoolAddress(ctx context.Context, log types.Log) ([]string, error) {
	poolAddresses, err := f.getPoolAddresses(ctx, log)
	if err != nil {
		return nil, err
	}
	return poolAddresses, nil
}

func (f *PoolFactory) getPoolAddresses(ctx context.Context, log types.Log) ([]string, error) {
	if len(log.Topics) == 0 {
		return nil, nil
	}

	if log.Address != f.controller {
		return nil, nil
	}

	switch log.Topics[0] {
	case controllerABI.Events["TokensTraded"].ID:
		e, err := controllerFilterer.ParseTokensTraded(log)
		if err != nil {
			return nil, nil
		}

		return []string{f.getPoolAddress(e.SourceToken, e.TargetToken)}, nil

	case controllerABI.Events["StrategyCreated"].ID:
		e, err := controllerFilterer.ParseStrategyCreated(log)
		if err != nil {
			return nil, err
		}

		return []string{f.getPoolAddress(e.Token0, e.Token1)}, nil

	case controllerABI.Events["StrategyDeleted"].ID:
		e, err := controllerFilterer.ParseStrategyDeleted(log)
		if err != nil {
			return nil, err
		}

		return []string{f.getPoolAddress(e.Token0, e.Token1)}, nil

	case controllerABI.Events["StrategyUpdated"].ID:
		e, err := controllerFilterer.ParseStrategyUpdated(log)
		if err != nil {
			return nil, err
		}

		return []string{f.getPoolAddress(e.Token0, e.Token1)}, nil

	case controllerABI.Events["PairTradingFeePPMUpdated"].ID:
		e, err := controllerFilterer.ParsePairTradingFeePPMUpdated(log)
		if err != nil {
			return nil, err
		}

		return []string{f.getPoolAddress(e.Token0, e.Token1)}, nil

	case controllerABI.Events["TradingFeePPMUpdated"].ID:
		pairs, err := getPairs(ctx, f.ethrpcClient, f.controller)
		if err != nil {
			return nil, err
		}

		poolAddresses := make([]string, 0, len(pairs))
		for _, pair := range pairs {
			poolAddresses = append(poolAddresses, f.getPoolAddress(pair[0], pair[1]))
		}

		return poolAddresses, nil

	default:
		return nil, nil
	}
}

func (f *PoolFactory) getPoolAddress(token0, token1 common.Address) string {
	return generatePoolAddress(f.controller, token0, token1)
}
