package solidlyv3

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/solidly-v3/abis"
)

var _ = poolfactory.RegisterFactoryC(DexTypeSolidlyV3, NewPoolFactory)

var (
	eventHashPoolCreated = solidlyV3FactoryABI.Events["PoolCreated"].ID
)

type PoolFactory struct {
	config              *Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			eventHashPoolCreated: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	p, err := factoryFilterer.ParsePoolCreated(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abis.FactoryPoolCreated, blockNumber uint64) (*entity.Pool, error) {
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

	swapFee, _ := p.Fee.Float64()

	extraBytes, err := json.Marshal(Extra{
		TickSpacing: p.TickSpacing.Uint64(),
	})
	if err != nil {
		return nil, err
	}

	staticExtraBytes, err := json.Marshal(uniswapv3.StaticExtra{
		PoolId: poolAddress,
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     poolAddress,
		SwapFee:     swapFee,
		Exchange:    f.config.DexID,
		Type:        DexTypeSolidlyV3,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      []*entity.PoolToken{&token0, &token1},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
		BlockNumber: blockNumber,
	}, nil
}
