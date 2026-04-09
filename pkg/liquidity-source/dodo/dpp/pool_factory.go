package dpp

import (
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dpp/abi"
	shared "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(PoolType, NewPoolFactory)

type PoolFactory struct {
	cfg                 *shared.Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *shared.Config) *PoolFactory {
	return &PoolFactory{
		cfg: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			factoryABI.Events["NewDPP"].ID: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	p, err := factoryFilterer.ParseNewDPP(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abis.DPPFactoryNewDPP, blockNumber uint64) (*entity.Pool, error) {
	poolAddress := strings.ToLower(hexutil.Encode(p.Dpp[:]))
	staticExtraBytes, err := json.Marshal(shared.StaticExtra{
		PoolId: poolAddress,
		Type:   shared.SubgraphPoolTypeDodoPrivate,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshal static extra data")
		return nil, err
	}

	return &entity.Pool{
		Address:   poolAddress,
		Exchange:  f.cfg.DexID,
		Type:      PoolType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{{
			Address:   strings.ToLower(hexutil.Encode(p.BaseToken[:])),
			Swappable: true,
		}, {
			Address:   strings.ToLower(hexutil.Encode(p.QuoteToken[:])),
			Swappable: true,
		}},
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNumber,
	}, nil
}
