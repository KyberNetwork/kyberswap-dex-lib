package liquiditybookv21

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(DexTypeLiquidityBookV21, NewPoolFactory)

type PoolFactory struct {
	config              *Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			factoryABI.Events["LBPairCreated"].ID:  {},
			factoryABI.Events["LBPairCreated0"].ID: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	switch event.Topics[0] {
	case factoryABI.Events["LBPairCreated"].ID:
		p, err := factoryFilterer.ParseLBPairCreated(event)
		if err != nil {
			return nil, err
		}
		return f.newPool(p.LBPair, p.TokenX, p.TokenY, event.BlockNumber)
	case factoryABI.Events["LBPairCreated0"].ID:
		p, err := factoryFilterer.ParseLBPairCreated0(event)
		if err != nil {
			return nil, err
		}
		return f.newPool(p.LBPair, p.TokenX, p.TokenY, event.BlockNumber)
	default:
		return nil, errors.New("event is not supported")
	}
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(pair, tokenX, tokenY common.Address, blockNbr uint64) (*entity.Pool, error) {
	token0 := entity.PoolToken{
		Address:   hexutil.Encode(tokenX[:]),
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   hexutil.Encode(tokenY[:]),
		Swappable: true,
	}

	return &entity.Pool{
		Address:     hexutil.Encode(pair[:]),
		Exchange:    f.config.DexID,
		Type:        DexTypeLiquidityBookV21,
		Timestamp:   time.Now().Unix(),
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{&token0, &token1},
		BlockNumber: blockNbr,
	}, nil
}
