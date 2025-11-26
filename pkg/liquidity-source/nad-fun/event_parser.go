package nadfun

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"

	pooldecode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/decode"
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

var _ = pooldecode.RegisterFactoryC(DexType, NewEventParser)

func NewEventParser(config *EventParserConfig) *EventParser {
	return &EventParser{
		config: config,
	}
}

func (p *EventParser) Decode(ctx context.Context, logs []types.Log) (map[string][]types.Log, error) {
	keys, err := p.GetKeys(ctx)
	if err != nil {
		return nil, err
	}
	bondingCurveAddress := keys[0]
	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		if !strings.EqualFold(log.Address.Hex(), bondingCurveAddress) {
			continue
		}
		switch log.Topics[0] {
		case bondingCurveABI.Events["CurveBuy"].ID,
			bondingCurveABI.Events["CurveSell"].ID:
			// topic 1: sender, topic 2: token
			if len(log.Topics) < 3 {
				break
			}
			token := common.HexToAddress(log.Topics[2].Hex()).Hex()
			poolAddress := GetPoolAddress(token)
			addressLogs[poolAddress] = append(addressLogs[poolAddress], log)

		case bondingCurveABI.Events["CurveSync"].ID,
			bondingCurveABI.Events["CurveTokenLocked"].ID,
			bondingCurveABI.Events["CurveGraduate"].ID:
			// topic 1: token
			if len(log.Topics) < 2 {
				break
			}
			token := common.HexToAddress(log.Topics[1].Hex()).Hex()
			poolAddress := GetPoolAddress(token)
			addressLogs[poolAddress] = append(addressLogs[poolAddress], log)
		}
	}

	return addressLogs, nil
}

func (p *EventParser) GetKeys(_ context.Context) ([]string, error) {
	if p.config.BondingCurve == "" {
		return nil, errors.New("bonding curve address is not set")
	}
	return []string{strings.ToLower(p.config.BondingCurve)}, nil
}
