package biconomyprop

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

// Metadata records the pool addresses already emitted. getPairs() returns every pair the venue
// serves, so no index cursor is needed.
type Metadata struct {
	Seen map[string]bool `json:"seen"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

type venueTokenPair struct {
	Token0 common.Address
	Token1 common.Address
}

// GetNewPools lists the venue's pairs, one pool per pair at the synthetic address
// <venue>_<token0>_<token1>.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}
	if metadata.Seen == nil {
		metadata.Seen = make(map[string]bool)
	}

	var pairs []venueTokenPair
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: venueABI, Target: u.config.Venue, Method: methodGetPairs}, []any{&pairs}).
		Call(); err != nil {
		return nil, metadataBytes, err
	}

	venue := strings.ToLower(u.config.Venue)
	staticExtra, err := json.Marshal(StaticExtra{Venue: venue})
	if err != nil {
		return nil, metadataBytes, err
	}

	var pools []entity.Pool
	for _, p := range pairs {
		token0 := hexutil.Encode(p.Token0[:])
		token1 := hexutil.Encode(p.Token1[:])
		addr := poolAddress(venue, token0, token1)
		if metadata.Seen[addr] {
			continue
		}
		metadata.Seen[addr] = true

		pools = append(pools, entity.Pool{
			Address:  addr,
			Exchange: u.config.DexID,
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: token0, Swappable: true},
				{Address: token1, Swappable: true},
			},
			StaticExtra: string(staticExtra),
		})
	}

	newMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMetadata, nil
}

func poolAddress(venue, token0, token1 string) string {
	return strings.ToLower(fmt.Sprintf("%s_%s_%s", venue, token0, token1))
}
