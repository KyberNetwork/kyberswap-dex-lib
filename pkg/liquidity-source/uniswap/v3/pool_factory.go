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

// Every DexType merged into this package registers the same NewPoolFactory, and every pool it
// creates is stamped Type: DexTypeUniswapV3 regardless of which one - the underlying logic is
// identical across all six, so any of the six registered types resolves the same simulator/
// tracker/factory. Exchange (Config.DexID) still carries the real per-deployment identity.
var (
	_ = poolfactory.RegisterFactoryC(DexTypeUniswapV3, NewPoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypePancakeV3, NewPoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypeRamsesV2, NewPoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypeSolidlyV3, NewPoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypeSlipstream, NewPoolFactory)
	_ = poolfactory.RegisterFactoryC(DexTypeNuriV2, NewPoolFactory)
)

// PoolFactory decodes PoolCreated events for every uniswap-v3 fork merged into this
// package. It needs no per-fork ABI: see the topic0 comment on poolCreatedEventIDWithFee
// in constant.go for why token0/token1/pool can always be read positionally.
type PoolFactory struct {
	config *Config
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{config: config}
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
		Type:        DexTypeUniswapV3,
		Timestamp:   time.Now().Unix(),
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{&t0, &t1},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
		BlockNumber: blockNumber,
	}, nil
}
