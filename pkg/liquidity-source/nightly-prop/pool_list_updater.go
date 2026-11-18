package nightlyprop

import (
	"context"
	"fmt"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	staticExtra  string
}

var _ = poollist.RegisterFactoryCE1(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolsListUpdater, error) {
	if !common.IsHexAddress(cfg.RouterAddress) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRouterAddress, cfg.RouterAddress)
	}
	if !common.IsHexAddress(cfg.PartnerAddress) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidPartnerAddress, cfg.PartnerAddress)
	}
	router, partner := common.HexToAddress(cfg.RouterAddress), common.HexToAddress(cfg.PartnerAddress)
	staticExtra, err := json.Marshal(StaticExtra{
		RouterAddress:  hexutil.Encode(router[:]),
		PartnerAddress: hexutil.Encode(partner[:]),
	})
	if err != nil {
		return nil, err
	}
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		staticExtra:  string(staticExtra),
	}, nil
}

// GetNewPools returns one pool for each route of the router that the cursor does not hold. The pool
// address is the NightlyPool proxy, and the tracker fills its reserves and state. On an error, it
// returns the old cursor, so the next call reads the same routes again.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var meta poolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			return nil, metadataBytes, fmt.Errorf("nightly-prop: decode metadata: %w", err)
		}
	}

	var routes []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: u.cfg.RouterAddress,
		Method: routerMethodGetRoutes,
	}, []any{&routes}).Call(); err != nil {
		return nil, metadataBytes, fmt.Errorf("nightly-prop: getRoutes: %w", err)
	}

	known := make(map[string]struct{}, len(meta.KnownPools))
	for _, pool := range meta.KnownPools {
		known[pool] = struct{}{}
	}
	listed := make([]string, len(routes))
	var added []string
	for i, route := range routes {
		listed[i] = hexutil.Encode(route[:])
		if _, ok := known[listed[i]]; !ok {
			added = append(added, listed[i])
		}
	}

	newMeta, err := json.Marshal(poolsListUpdaterMetadata{KnownPools: listed})
	if err != nil {
		return nil, metadataBytes, err
	}
	if len(added) == 0 {
		return nil, newMeta, nil
	}

	tokens := make([]poolTokens, len(added))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, pool := range added {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: pool,
			Method: poolMethodGetTokens,
		}, []any{&tokens[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, metadataBytes, fmt.Errorf("nightly-prop: getTokens: %w", err)
	}

	now := time.Now().Unix()
	pools := make([]entity.Pool, len(added))
	for i, pool := range added {
		pools[i] = entity.Pool{
			Address:   pool,
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(tokens[i].Base[:]), Swappable: true},
				{Address: hexutil.Encode(tokens[i].Quote[:]), Swappable: true},
			},
			StaticExtra: u.staticExtra,
		}
	}
	return pools, newMeta, nil
}
