package carbon

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	pooldecode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/decode"
)

var _ = pooldecode.RegisterFactoryCE(DexType, NewEventParser)

type EventParser struct {
	ethrpcClient *ethrpc.Client
	controller   common.Address
}

func NewEventParser(config *Config, ethrpcClient *ethrpc.Client) *EventParser {
	return &EventParser{
		ethrpcClient: ethrpcClient,
		controller:   config.Controller,
	}
}

func (ep *EventParser) Decode(ctx context.Context, logs []types.Log) (map[string][]types.Log, error) {
	addressLogs := make(map[string][]types.Log)

	for _, log := range logs {
		poolAddresses, err := ep.getPoolAddresses(ctx, log)
		if err != nil {
			return nil, err
		}

		for _, poolAddress := range poolAddresses {
			if poolAddress != "" {
				addressLogs[poolAddress] = append(addressLogs[poolAddress], log)
			}
		}
	}

	return addressLogs, nil
}

func (ep *EventParser) getPoolAddresses(ctx context.Context, log types.Log) ([]string, error) {
	if len(log.Topics) == 0 {
		return nil, nil
	}

	if log.Address != ep.controller {
		return nil, nil
	}

	switch log.Topics[0] {
	case controllerABI.Events["TokensTraded"].ID:
		e, err := controllerFilterer.ParseTokensTraded(log)
		if err != nil {
			return nil, nil
		}

		return []string{ep.getPoolAddress(e.SourceToken, e.TargetToken)}, nil

	case controllerABI.Events["StrategyCreated"].ID:
		e, err := controllerFilterer.ParseStrategyCreated(log)
		if err != nil {
			return nil, err
		}

		return []string{ep.getPoolAddress(e.Token0, e.Token1)}, nil

	case controllerABI.Events["StrategyDeleted"].ID:
		e, err := controllerFilterer.ParseStrategyDeleted(log)
		if err != nil {
			return nil, err
		}

		return []string{ep.getPoolAddress(e.Token0, e.Token1)}, nil

	case controllerABI.Events["StrategyUpdated"].ID:
		e, err := controllerFilterer.ParseStrategyUpdated(log)
		if err != nil {
			return nil, err
		}

		return []string{ep.getPoolAddress(e.Token0, e.Token1)}, nil

	case controllerABI.Events["PairTradingFeePPMUpdated"].ID:
		e, err := controllerFilterer.ParsePairTradingFeePPMUpdated(log)
		if err != nil {
			return nil, err
		}

		return []string{ep.getPoolAddress(e.Token0, e.Token1)}, nil

	case controllerABI.Events["TradingFeePPMUpdated"].ID:
		pairs, err := getPairs(ctx, ep.ethrpcClient, ep.controller)
		if err != nil {
			return nil, err
		}

		poolAddresses := make([]string, 0, len(pairs))
		for _, pair := range pairs {
			poolAddresses = append(poolAddresses, ep.getPoolAddress(pair[0], pair[1]))
		}

		return poolAddresses, nil

	default:
		return nil, nil
	}
}

func (ep *EventParser) GetKeys(_ context.Context) ([]string, error) {
	return []string{hexutil.Encode(ep.controller[:])}, nil
}

func (ep *EventParser) getPoolAddress(token0, token1 common.Address) string {
	return generatePoolAddress(ep.controller, token0, token1)
}
