package syntropiaprop

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

// Metadata records the pools already emitted. getPairs() returns every listed pair, and the
// owner can only add to it, so no cursor is needed.
type Metadata struct {
	Seen map[string]bool `json:"seen"`
}

var _ = poollist.RegisterFactoryCE1(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolsListUpdater, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}, nil
}

// GetNewPools lists the venue's pairs, one pool per pair at the synthetic address
// <pool>_<token0>_<token1>.
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

	var pairs []TokenPair
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: u.config.Pool, Method: "getPairs"}, []any{&pairs}).
		Call(); err != nil {
		return nil, metadataBytes, err
	}

	pools, err := u.buildPools(pairs, &metadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMetadata, nil
}

// buildPools returns the pairs not emitted before and marks them as seen.
func (u *PoolsListUpdater) buildPools(pairs []TokenPair, metadata *Metadata) ([]entity.Pool, error) {
	staticExtra, err := json.Marshal(StaticExtra{Pool: u.config.Pool})
	if err != nil {
		return nil, err
	}

	var pools []entity.Pool
	for _, pair := range pairs {
		token0, token1 := hexutil.Encode(pair.Token0[:]), hexutil.Encode(pair.Token1[:])
		address := fmt.Sprintf("%s_%s_%s", u.config.Pool, token0, token1)
		if metadata.Seen[address] {
			continue
		}
		metadata.Seen[address] = true

		pools = append(pools, entity.Pool{
			Address:  address,
			Exchange: u.config.DexID,
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: token0, Swappable: true},
				{Address: token1, Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
		})
	}
	return pools, nil
}
