package printr

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"

	pooldecode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/decode"
)

var _ = pooldecode.RegisterFactoryC(DexType, NewEventParser)

type EventParser struct {
	printrAddr      string
	supportedEvents map[common.Hash]int
}

func NewEventParser(cfg *Config) *EventParser {
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
	keys, err := p.GetKeys(ctx)
	if err != nil {
		return nil, err
	}
	printrAddr := keys[0]

	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		if !strings.EqualFold(log.Address.Hex(), printrAddr) {
			continue
		}
		if len(log.Topics) < 2 {
			continue
		}

		tokenTopicIndex, ok := p.supportedEvents[log.Topics[0]]
		if !ok {
			continue
		}

		if len(log.Topics) <= tokenTopicIndex {
			continue
		}

		token := common.HexToAddress(log.Topics[tokenTopicIndex].Hex()).Hex()
		poolAddress := strings.ToLower(token) // pool address == token address
		addressLogs[poolAddress] = append(addressLogs[poolAddress], log)
	}

	return addressLogs, nil
}

func (p *EventParser) GetKeys(_ context.Context) ([]string, error) {
	if p.printrAddr == "" {
		return nil, errors.New("printr address is not set")
	}
	return []string{strings.ToLower(p.printrAddr)}, nil
}
