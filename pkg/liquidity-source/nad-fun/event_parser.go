package nadfun

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

const (
	prefix    = "bc"
	seperator = "-"
)

func GetPoolAddress(token string) string {
	return strings.ToLower(prefix + seperator + token)
}

type EventParserConfig struct {
	BondingCurve string `json:"bondingCurve,omitempty"`
}

type EventParser struct {
	config *EventParserConfig
}

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

func NewPoolFactory(config *EventParserConfig) *EventParser {
	return &EventParser{
		config: config,
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
			addressLogs[poolAddress] = append(addressLogs[poolAddress], log)
		}
	}

	return addressLogs, nil
}

func (p *EventParser) DecodePoolAddressesFromFactoryLog(ctx context.Context, log types.Log) ([]string, error) {
	if !strings.EqualFold(log.Address.Hex(), p.config.BondingCurve) {
		return nil, nil
	}
	switch log.Topics[0] {
	case bondingCurveABI.Events["CurveBuy"].ID,
		bondingCurveABI.Events["CurveSell"].ID:
		// topic 1: sender, topic 2: token
		if len(log.Topics) < 3 {
			break
		}
		token := common.HexToAddress(log.Topics[2].Hex()).Hex()
		return []string{GetPoolAddress(token)}, nil

	case bondingCurveABI.Events["CurveSync"].ID,
		bondingCurveABI.Events["CurveTokenLocked"].ID,
		bondingCurveABI.Events["CurveGraduate"].ID:
		// topic 1: token
		if len(log.Topics) < 2 {
			break
		}
		token := common.HexToAddress(log.Topics[1].Hex()).Hex()
		return []string{GetPoolAddress(token)}, nil
	}
	return nil, nil
}

func (p *EventParser) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	// TODO: Implement this (non tick-based pool creation)
	return nil, nil
}

func (p *EventParser) IsEventSupported(event common.Hash) bool {
	// TODO: Implement this (non tick-based pool creation)
	return true
}
