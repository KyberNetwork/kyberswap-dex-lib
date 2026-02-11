package nadfun

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type PoolFactory struct {
	cfg                 *Config
	bondingCurveAddress string
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		cfg:                 config,
		bondingCurveAddress: config.BondingCurveAddress,
	}
}

func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	return nil, nil
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	return true
}

func (f *PoolFactory) DecodePoolAddress(ctx context.Context, log types.Log) ([]string, error) {
	if !strings.EqualFold(log.Address.Hex(), f.bondingCurveAddress) {
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
