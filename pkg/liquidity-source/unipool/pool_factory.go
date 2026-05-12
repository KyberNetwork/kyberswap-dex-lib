package unipool

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type PoolFactory struct {
	config *Config
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{config: config}
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	return event == uniPoolFactoryABI.Events[factoryEventPairCreated].ID
}

func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	token0, token1, pair, err := f.parsePairCreated(event)
	if err != nil {
		return nil, err
	}
	return f.newPool(token0, token1, pair, event.BlockNumber)
}

// DecodePoolAddressesFromFactoryLog returns the pool address(es) touched by a
// factory log without building the full entity.Pool. Used by indexer services
// that only need to know which pools need re-tracking after a block.
func (f *PoolFactory) DecodePoolAddressesFromFactoryLog(_ context.Context, event types.Log) ([]string, error) {
	_, _, pair, err := f.parsePairCreated(event)
	if err != nil {
		return nil, err
	}
	return []string{hexutil.Encode(pair[:])}, nil
}

// parsePairCreated decodes a PairCreated event:
//
//	event PairCreated(address indexed token0, address indexed token1, address pair, uint256 totalPairs)
func (f *PoolFactory) parsePairCreated(log types.Log) (token0, token1, pair common.Address, err error) {
	if len(log.Topics) != 3 ||
		eth.IsZeroAddress(log.Address) ||
		!strings.EqualFold(log.Address.Hex(), f.config.FactoryAddress) ||
		log.Topics[0] != uniPoolFactoryABI.Events[factoryEventPairCreated].ID {
		err = errors.New("event is not supported")
		return
	}

	token0 = common.BytesToAddress(log.Topics[1].Bytes())
	token1 = common.BytesToAddress(log.Topics[2].Bytes())

	unpacked, err := uniPoolFactoryABI.Events[factoryEventPairCreated].Inputs.NonIndexed().Unpack(log.Data)
	if err != nil {
		return
	}
	if len(unpacked) < 1 {
		err = errors.New("malformed PairCreated event data")
		return
	}
	addr, ok := unpacked[0].(common.Address)
	if !ok {
		err = errors.New("malformed PairCreated event pair field")
		return
	}
	pair = addr
	return
}

func (f *PoolFactory) newPool(token0, token1, pair common.Address, blockNumber uint64) (*entity.Pool, error) {
	staticExtra := StaticExtra{FactoryAddress: f.config.FactoryAddress}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, err
	}

	extra := zeroExtra()
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   hexutil.Encode(pair[:]),
		Exchange:  f.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(token0[:]), Swappable: true},
			{Address: hexutil.Encode(token1[:]), Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNumber,
	}, nil
}
