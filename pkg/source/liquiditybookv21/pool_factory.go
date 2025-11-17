package liquiditybookv21

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv21/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(DexTypeLiquidityBookV21, NewPoolFactory)

var (
	eventLBPairCreated = factoryABI.Events["LBPairCreated"].ID
)

type PoolFactory struct {
	config              *Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			eventLBPairCreated: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	p, err := factoryFilterer.ParseLBPairCreated(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abis.LBFactoryLBPairCreated, blockNbr uint64) (*entity.Pool, error) {
	token0 := entity.PoolToken{
		Address:   hexutil.Encode(p.TokenX[:]),
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   hexutil.Encode(p.TokenY[:]),
		Swappable: true,
	}

	return &entity.Pool{
		Address:     hexutil.Encode(p.LBPair[:]),
		Exchange:    f.config.DexID,
		Type:        DexTypeLiquidityBookV21,
		Timestamp:   time.Now().Unix(),
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{&token0, &token1},
		BlockNumber: blockNbr,
	}, nil
}
