package integral

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral/abis"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

var (
	eventHashPoolCreated       = factoryABI.Events["Pool"].ID
	eventHashCustomPoolCreated = factoryABI.Events["CustomPool"].ID
)

type PoolFactory struct {
	config              *Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			eventHashPoolCreated:       {},
			eventHashCustomPoolCreated: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
		return nil, errors.New("event is not supported")
	}

	switch event.Topics[0] {
	case eventHashPoolCreated:
		pool, err := factoryFilterer.ParsePool(event)
		if err != nil {
			return nil, err
		}
		return f.newPool(pool, event.BlockNumber)
	case eventHashCustomPoolCreated:
		customPool, err := factoryFilterer.ParseCustomPool(event)
		if err != nil {
			return nil, err
		}
		return f.newCustomPool(customPool, event.BlockNumber)
	default:
		return nil, errors.New("event is not supported")
	}
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abis.FactoryPool, blockNbr uint64) (*entity.Pool, error) {
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

	staticExtraBytes, err := json.Marshal(uniswapv3.StaticExtra{
		PoolId: poolAddress,
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     poolAddress,
		Exchange:    f.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      []*entity.PoolToken{&token0, &token1},
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNbr,
	}, nil
}

func (f *PoolFactory) newCustomPool(p *abis.FactoryCustomPool, blockNbr uint64) (*entity.Pool, error) {
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

	staticExtraBytes, err := json.Marshal(uniswapv3.StaticExtra{
		PoolId: poolAddress,
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     poolAddress,
		Exchange:    f.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      []*entity.PoolToken{&token0, &token1},
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNbr,
	}, nil
}
