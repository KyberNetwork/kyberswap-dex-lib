package lunya

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactoryDecoder)

// PoolFactoryDecoder turns LunyaPoolFactory's PoolCreated into a pool from the log alone, whatever its
// type. Reserves and swap state are left zero for PoolTracker, which pool-service runs as soon as the
// pool is saved.
//
// LunyaPoolFactory can also emit a Uniswap V3-shaped PoolCreated(address,address,uint24,int24,address)
// for indexers that only know that event, with the pool type in the fee slot. It is the same pool as
// the Lunya event next to it, so only the Lunya event is decoded.
type PoolFactoryDecoder struct {
	config *Config
}

func NewPoolFactoryDecoder(cfg *Config) *PoolFactoryDecoder {
	return &PoolFactoryDecoder{config: cfg}
}

func (d *PoolFactoryDecoder) IsEventSupported(hash common.Hash) bool {
	return hash == poolCreatedEvent.ID
}

func (d *PoolFactoryDecoder) SupportedEventTopics() []common.Hash {
	return []common.Hash{poolCreatedEvent.ID}
}

func (d *PoolFactoryDecoder) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	if !strings.EqualFold(event.Address.Hex(), d.config.Factory) {
		return nil, nil
	}
	if len(event.Topics) != 4 || event.Topics[0] != poolCreatedEvent.ID {
		return nil, ErrMalformedLog
	}

	poolType := event.Topics[3].Big()
	if !poolType.IsUint64() || poolType.Uint64() > poolTypeStable {
		// a type this source does not know how to price
		return nil, nil
	}

	values, err := poolCreatedEvent.Inputs.NonIndexed().Unpack(event.Data)
	if err != nil {
		return nil, err
	}
	poolAddress, ok := values[2].(common.Address)
	if !ok {
		return nil, ErrMalformedLog
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{PoolType: uint8(poolType.Uint64())})
	if err != nil {
		return nil, err
	}

	token0 := common.BytesToAddress(event.Topics[1].Bytes())
	token1 := common.BytesToAddress(event.Topics[2].Bytes())

	return &entity.Pool{
		Address:   hexutil.Encode(poolAddress[:]),
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(token0[:]), Swappable: true},
			{Address: hexutil.Encode(token1[:]), Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		BlockNumber: event.BlockNumber,
	}, nil
}
