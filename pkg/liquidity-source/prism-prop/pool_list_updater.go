package prismprop

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools re-lists every pair on every run: getSupportedPairs is a single
// cheap call, and prism-prop has no pair-added event to track a cursor from
// (see titan-prop's KnownVenues cursor for the pattern this would follow if
// that changes). The pool-service layer dedupes by address, so re-listing an
// already-known pair is a no-op.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	var pairs []Pair
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: u.config.RouterAddress,
		Method: methodGetSupportedPairs,
	}, []any{&pairs}).Aggregate(); err != nil {
		return nil, nil, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{RouterAddress: strings.ToLower(u.config.RouterAddress)})
	if err != nil {
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0, len(pairs))
	for _, pair := range pairs {
		pools = append(pools, entity.Pool{
			Address:  poolAddress(u.config.RouterAddress, pair.Token0.Hex(), pair.Token1.Hex()),
			Exchange: u.config.DexID,
			Type:     DexType,
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(pair.Token0.Hex()), Swappable: true},
				{Address: strings.ToLower(pair.Token1.Hex()), Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
			Timestamp:   time.Now().Unix(),
		})
	}
	return pools, nil, nil
}

// poolAddress is synthetic: prism-prop has one router contract quoting every
// pair, so the "pool address" is derived from (router, pair), same as
// titan-prop's IPropAMM venues.
func poolAddress(router, token0, token1 string) string {
	return strings.ToLower(router + "_" + token0 + "_" + token1)
}
