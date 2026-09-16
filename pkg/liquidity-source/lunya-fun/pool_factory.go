package lunyafun

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

// PoolFactoryDecoder turns LunyaLaunchFactory's LaunchCreated into a pool from the log alone: the launch
// contract is the pool, and it trades its own token against the quote token named there. Curve state is
// left to PoolTracker, which pool-service runs as soon as the pool is saved.
type PoolFactoryDecoder struct {
	config *Config
}

func NewPoolFactoryDecoder(cfg *Config) *PoolFactoryDecoder {
	return &PoolFactoryDecoder{config: cfg}
}

func (d *PoolFactoryDecoder) IsEventSupported(hash common.Hash) bool {
	return hash == launchCreatedEvent.ID
}

func (d *PoolFactoryDecoder) SupportedEventTopics() []common.Hash {
	return []common.Hash{launchCreatedEvent.ID}
}

func (d *PoolFactoryDecoder) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	if !strings.EqualFold(event.Address.Hex(), d.config.Factory) {
		return nil, nil
	}
	if len(event.Topics) != 4 || event.Topics[0] != launchCreatedEvent.ID {
		return nil, ErrMalformedLog
	}

	values, err := launchCreatedEvent.Inputs.NonIndexed().Unpack(event.Data)
	if err != nil {
		return nil, err
	}
	launchType, ok := values[0].(uint8)
	if !ok {
		return nil, ErrMalformedLog
	}
	quoteToken, ok := values[1].(common.Address)
	if !ok {
		return nil, ErrMalformedLog
	}
	if launchType != launchTypeCP {
		// a curve this source does not know how to price
		return nil, nil
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{LaunchType: launchType})
	if err != nil {
		return nil, err
	}

	launch := common.BytesToAddress(event.Topics[1].Bytes())
	token := common.BytesToAddress(event.Topics[2].Bytes())

	return &entity.Pool{
		Address:   hexutil.Encode(launch[:]),
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(quoteToken[:]), Swappable: true},
			{Address: hexutil.Encode(token[:]), Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		BlockNumber: event.BlockNumber,
	}, nil
}
