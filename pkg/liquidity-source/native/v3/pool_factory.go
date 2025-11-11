package v3

import (
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

var (
	eventHashPoolCreated = factoryABI.Events["PoolCreated"].ID
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

	staticExtraBytes, err := json.Marshal(StaticExtra{
		TickSpacing: p.TickSpacing.Uint64(),
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     poolAddress,
		Exchange:    f.config.DexID,
		Type:        DexType,
		SwapFee:     swapFee,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      []*entity.PoolToken{&token0, &token1},
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNumber,
	}, nil
}
