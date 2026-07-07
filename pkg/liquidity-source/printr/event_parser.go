package printr

import (
	"context"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type EventParser struct {
	printrAddr      string
	supportedEvents map[common.Hash]int
}

func NewPoolFactory(cfg *Config) *EventParser {
	supported := map[common.Hash]int{}

	add := func(eventName string, tokenTopicIndex int) {
		if ev, ok := printrABI.Events[eventName]; ok {
			supported[ev.ID] = tokenTopicIndex
		}
	}

	add("TokenTrade", 1)     // topic1: token
	add("TokenGraduated", 1) // topic1: token

	return &EventParser{
		printrAddr:      strings.ToLower(cfg.PrintrAddr),
		supportedEvents: supported,
	}
}

func (p *EventParser) Decode(ctx context.Context, logs []types.Log) (map[string][]types.Log, error) {
	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		poolAddresses, err := p.DecodePoolAddressesFromFactoryLog(ctx, log)
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

func (p *EventParser) DecodePoolAddressesFromFactoryLog(ctx context.Context, log types.Log) ([]string, error) {
	if !strings.EqualFold(log.Address.Hex(), p.printrAddr) || len(log.Topics) < 2 {
		return nil, nil
	}

	tokenTopicIndex, ok := p.supportedEvents[log.Topics[0]]
	if !ok || len(log.Topics) <= tokenTopicIndex {
		return nil, nil
	}
	token := common.HexToAddress(log.Topics[tokenTopicIndex].Hex()).Hex()
	return []string{strings.ToLower(token)}, nil
}

func (ep *EventParser) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	// TODO: Implement this (non tick-based pool creation)
	return nil, errors.New("not implemented")
}

func (ep *EventParser) IsEventSupported(event common.Hash) bool {
	// TODO: Implement this (non tick-based pool creation)
	return false
}
