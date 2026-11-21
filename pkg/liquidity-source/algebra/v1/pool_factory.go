package algebrav1

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/v1/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(DexTypeAlgebraV1, NewPoolFactory)

type PoolFactory struct {
	config              *Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			algebraV1FactoryABI.Events["Pool"].ID: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	p, err := factoryFilterer.ParsePool(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abis.FactoryPool, blockNumber uint64) (*entity.Pool, error) {
	poolAddress := hexutil.Encode(p.Pool[:])

	token0 := entity.PoolToken{
		Address:   hexutil.Encode(p.Token0[:]),
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   hexutil.Encode(p.Token1[:]),
		Swappable: true,
	}
	reserves := entity.PoolReserves{
		"0", "0",
	}

	return &entity.Pool{
		Address:     poolAddress,
		Exchange:    f.config.DexID,
		Type:        DexTypeAlgebraV1,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      []*entity.PoolToken{&token0, &token1},
		BlockNumber: blockNumber,
	}, nil
}
