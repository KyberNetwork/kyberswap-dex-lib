package curve

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolFactoryDecoder turns ArcadeHook's LaunchCreated into a curve pool, from the
// log alone. Only PUMP launches have a bonding curve; CLANKER and RWA launches are
// live in their uniswap-v4 pool from birth (see hooks/arcade) and are skipped.
// State is filled by PoolTracker on its first pass.
type PoolFactoryDecoder struct {
	config *Config
}

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactoryDecoder)

func NewPoolFactoryDecoder(cfg *Config) *PoolFactoryDecoder {
	return &PoolFactoryDecoder{config: cfg}
}

func (d *PoolFactoryDecoder) IsEventSupported(hash common.Hash) bool {
	return hash == launchCreatedEventHash
}

func (d *PoolFactoryDecoder) SupportedEventTopics() []common.Hash {
	return []common.Hash{launchCreatedEventHash}
}

func (d *PoolFactoryDecoder) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	if !strings.EqualFold(event.Address.Hex(), d.config.Hook) {
		return nil, nil
	}
	if len(event.Topics) != 3 || event.Topics[0] != launchCreatedEventHash {
		return nil, nil
	}
	values, err := arcadeHookABI.Events["LaunchCreated"].Inputs.NonIndexed().Unpack(event.Data)
	if err != nil {
		return nil, err
	}
	if len(values) != 2 {
		return nil, ErrInvalidToken
	}
	mode, ok := values[1].(uint8)
	if !ok {
		return nil, ErrInvalidToken
	}
	if mode != modePump {
		return nil, nil
	}

	poolID := event.Topics[1].Hex()
	token := common.BytesToAddress(event.Topics[2].Bytes())
	usdc := valueobject.WrappedNativeMap[d.config.ChainID]

	staticExtra, err := json.Marshal(StaticExtra{Hook: strings.ToLower(d.config.Hook)})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   PoolAddress(poolID),
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: strings.ToLower(usdc), Swappable: true},
			{Address: hexutil.Encode(token[:]), Swappable: true},
		},
		StaticExtra: string(staticExtra),
		Extra:       "{}",
	}, nil
}

// DecodePoolAddressesFromFactoryLog routes the hook's per-launch curve events
// (all indexed by PoolId) to the launch's pool, so live trades refresh its state.
func (d *PoolFactoryDecoder) DecodePoolAddressesFromFactoryLog(_ context.Context, log types.Log) ([]string, error) {
	if !strings.EqualFold(log.Address.Hex(), d.config.Hook) || len(log.Topics) < 2 {
		return nil, nil
	}
	switch log.Topics[0] {
	case curveBuyEventHash, curveSellEventHash, graduatedEventHash:
		return []string{PoolAddress(log.Topics[1].Hex())}, nil
	}
	return nil, nil
}
