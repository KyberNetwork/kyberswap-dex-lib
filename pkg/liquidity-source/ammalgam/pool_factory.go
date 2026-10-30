package ammalgam

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var (
	_                   = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)
	errInvalidPairEvent = errors.New("invalid Ammalgam PairCreated event")
)

type PoolFactory struct {
	config *Config
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{config: config}
}

func (f *PoolFactory) SupportedEventTopics() []common.Hash {
	return []common.Hash{factoryABI.Events["PairCreated"].ID}
}

func (f *PoolFactory) IsEventSupported(topic common.Hash) bool {
	return topic == factoryABI.Events["PairCreated"].ID
}

func (f *PoolFactory) DecodePoolAddressesFromFactoryLog(_ context.Context, event ethtypes.Log) ([]string, error) {
	pair, err := f.DecodePoolCreated(event)
	if err != nil {
		return nil, err
	}
	return []string{pair.Address}, nil
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	if f == nil || f.config == nil || !validPairAddress(f.config.FactoryAddress) ||
		!strings.EqualFold(event.Address.Hex(), f.config.FactoryAddress) ||
		len(event.Topics) != 3 || !f.IsEventSupported(event.Topics[0]) || len(event.Data) != 64 {
		return nil, errInvalidPairEvent
	}
	// Indexed address topics and the ABI-encoded pair address have twelve zero
	// prefix bytes. Reject malformed words instead of silently truncating them.
	for _, word := range [][]byte{event.Topics[1][:], event.Topics[2][:], event.Data[:32]} {
		for _, b := range word[:12] {
			if b != 0 {
				return nil, errInvalidPairEvent
			}
		}
	}
	tokenX := common.BytesToAddress(event.Topics[1][12:])
	tokenY := common.BytesToAddress(event.Topics[2][12:])
	pair := common.BytesToAddress(event.Data[12:32])
	if !validPairAddress(tokenX.Hex()) || !validPairAddress(tokenY.Hex()) ||
		!validPairAddress(pair.Hex()) || bytes.Compare(tokenX[:], tokenY[:]) >= 0 {
		return nil, errInvalidPairEvent
	}

	staticExtra, err := json.Marshal(StaticExtra{
		FactoryAddress:     strings.ToLower(f.config.FactoryAddress),
		PairBeacon:         strings.ToLower(f.config.PairBeacon),
		PairImplementation: strings.ToLower(f.config.PairImplementation),
		Gas:                f.config.Gas,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal Ammalgam static extra: %w", err)
	}
	extra, err := json.Marshal(Extra{})
	if err != nil {
		return nil, fmt.Errorf("marshal Ammalgam extra: %w", err)
	}
	return &entity.Pool{
		Address:     hexutil.Encode(pair[:]),
		Exchange:    f.config.DexID,
		Type:        DexType,
		BlockNumber: event.BlockNumber,
		Timestamp:   time.Now().Unix(),
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(tokenX[:]), Swappable: true},
			{Address: hexutil.Encode(tokenY[:]), Swappable: true},
		},
		Reserves:    entity.PoolReserves{"0", "0"},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	}, nil
}
