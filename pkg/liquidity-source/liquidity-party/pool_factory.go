package liquidityparty

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

// PoolFactory is the primary pool-discovery path: it decodes PartyPlanner's PartyStarted event so a
// pool appears the block it is created, with no index poll. The getAllPools paging in
// pool_list_updater.go remains only as the cold-start backfill for pools created before the event
// subscription began (pools are admin-created and never leave the index).
type PoolFactory struct {
	config *Config
}

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{config: config}
}

// IsEventSupported reports whether topic0 is PartyPlanner's PartyStarted event, so pool-service
// subscribes to it and routes matching logs into DecodePoolCreated.
func (f *PoolFactory) IsEventSupported(topic common.Hash) bool {
	return topic == partyStartedEventTopic
}

// DecodePoolCreated parses PartyStarted(pool indexed, name, symbol, tokens[]) into a fresh
// entity.Pool. Per repo rules it sets only {Address, Swappable:true} per token with "0" reserve
// placeholders; the tracker fills swap state and reserves on the next refresh. The pool address
// (topic1) and token addresses are lowercased. No StaticExtra/Extra is emitted so the tracker's
// cold-start killed() backstop still runs once for the new pool (see pool_tracker.go).
func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	if len(event.Topics) < 2 {
		return nil, ErrInvalidEvent
	}
	poolAddress := strings.ToLower(common.BytesToAddress(event.Topics[1].Bytes()).Hex())

	unpacked := make(map[string]any)
	if err := partyPlannerABI.UnpackIntoMap(unpacked, plannerEventPartyStarted, event.Data); err != nil {
		return nil, err
	}
	rawTokens, ok := unpacked["tokens"].([]common.Address)
	if !ok || len(rawTokens) == 0 {
		return nil, ErrInvalidEvent
	}

	tokens := make([]*entity.PoolToken, len(rawTokens))
	reserves := make(entity.PoolReserves, len(rawTokens))
	for i, t := range rawTokens {
		tokens[i] = &entity.PoolToken{Address: strings.ToLower(t.Hex()), Swappable: true}
		reserves[i] = "0"
	}

	return &entity.Pool{
		Address:     poolAddress,
		Exchange:    f.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		BlockNumber: event.BlockNumber,
	}, nil
}
