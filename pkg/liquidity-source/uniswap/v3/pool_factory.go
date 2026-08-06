package uniswapv3

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var (
	_ = poolfactory.RegisterFactoryC(DexTypeUniswapV3, NewUniswapV3PoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypePancakeV3, NewPancakeV3PoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypeRamsesV2, NewRamsesV2PoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypeSolidlyV3, NewSolidlyV3PoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypeSlipstream, NewSlipstreamPoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypeNuriV2, NewNuriV2PoolFactory)
)

func NewUniswapV3PoolFactory(config *Config) *PoolFactory {
	return newPoolFactory(config, DexTypeUniswapV3)
}

func NewPancakeV3PoolFactory(config *Config) *PoolFactory {
	return newPoolFactory(config, DexTypePancakeV3)
}

func NewRamsesV2PoolFactory(config *Config) *PoolFactory {
	return newPoolFactory(config, DexTypeRamsesV2)
}

func NewSolidlyV3PoolFactory(config *Config) *PoolFactory {
	return newPoolFactory(config, DexTypeSolidlyV3)
}

func NewSlipstreamPoolFactory(config *Config) *PoolFactory {
	return newPoolFactory(config, DexTypeSlipstream)
}

func NewNuriV2PoolFactory(config *Config) *PoolFactory {
	return newPoolFactory(config, DexTypeNuriV2)
}

// PoolFactory decodes PoolCreated events for every uniswap-v3 fork merged into this
// package. It needs no per-fork ABI: see the topic0 comment on poolCreatedEventIDWithFee
// in constant.go for why token0/token1/pool can always be read positionally.
type PoolFactory struct {
	config  *Config
	dexType string
}

func newPoolFactory(config *Config, dexType string) *PoolFactory {
	return &PoolFactory{config: config, dexType: dexType}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	if len(event.Topics) < 3 || len(event.Data) < 32 {
		return nil, ErrMalformedLog
	}

	token0 := common.BytesToAddress(event.Topics[1].Bytes())
	token1 := common.BytesToAddress(event.Topics[2].Bytes())
	poolAddress := common.BytesToAddress(event.Data[len(event.Data)-32:])

	return f.newPool(token0, token1, poolAddress, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := poolCreatedEventIDs[event]
	return ok
}

func (f *PoolFactory) newPool(token0, token1, poolAddress common.Address, blockNumber uint64) (*entity.Pool, error) {
	poolAddressHex := hexutil.Encode(poolAddress[:])

	t0 := entity.PoolToken{Address: hexutil.Encode(token0[:]), Swappable: true}
	t1 := entity.PoolToken{Address: hexutil.Encode(token1[:]), Swappable: true}

	// fee and tickSpacing are intentionally left zero here: the tracker's first refresh
	// (which must run before this pool has any ticks anyway) fills both in from the live
	// contract, generically, regardless of which fork this pool belongs to.
	extraBytes, err := json.Marshal(ExtraTickU256{})
	if err != nil {
		return nil, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{PoolId: poolAddressHex})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     poolAddressHex,
		Exchange:    f.config.DexID,
		Type:        f.dexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{&t0, &t1},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
		BlockNumber: blockNumber,
	}, nil
}
