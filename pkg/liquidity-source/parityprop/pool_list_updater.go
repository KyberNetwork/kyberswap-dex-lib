package parityprop

import (
	"context"
	"math/big"
	"time"

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

// Metadata persists the set of pool addresses already resolved into
// entity.Pool, keyed by address rather than a simple count: PmmRegistry's
// own removePool() is documented as swap-and-pop ("ordering of getPools()
// is not guaranteed stable"), so an index-based offset could permanently
// skip a pool that gets swapped into an already-scanned slot.
type Metadata struct {
	Seen map[string]bool `json:"seen"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools discovers pools via PmmRegistry.getPools(). Reserves are left
// at "0" -- the tracker fills real reserves on the pool's first
// GetNewPoolState refresh.
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

	var poolAddrs []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: pmmRegistryABI, Target: u.config.Registry, Method: methodGetPools}, []any{&poolAddrs}).
		Call(); err != nil {
		return nil, metadataBytes, err
	}

	newAddrs := unseenAddrs(poolAddrs, metadata.Seen)
	if len(newAddrs) == 0 {
		return nil, metadataBytes, nil
	}

	pools, err := u.resolvePools(ctx, newAddrs)
	if err != nil {
		return nil, metadataBytes, err
	}

	for _, addr := range newAddrs {
		metadata.Seen[hexutil.Encode(addr[:])] = true
	}
	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

// unseenAddrs filters poolAddrs down to those not already marked in seen,
// by address rather than position -- see Metadata's doc comment for why an
// index-based cutoff is unsound against this registry.
func unseenAddrs(poolAddrs []common.Address, seen map[string]bool) []common.Address {
	newAddrs := make([]common.Address, 0, len(poolAddrs))
	for _, addr := range poolAddrs {
		if !seen[hexutil.Encode(addr[:])] {
			newAddrs = append(newAddrs, addr)
		}
	}
	return newAddrs
}

func (u *PoolsListUpdater) resolvePools(ctx context.Context, addrs []common.Address) ([]entity.Pool, error) {
	n := len(addrs)
	baseHx := make([]common.Address, n)
	quoteHx := make([]common.Address, n)
	baseScaleRaw := make([]*big.Int, n)
	quoteScaleRaw := make([]*big.Int, n)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range addrs {
		target := hexutil.Encode(addr[:])
		req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: target, Method: methodBase}, []any{&baseHx[i]})
		req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: target, Method: methodQuote}, []any{&quoteHx[i]})
		req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: target, Method: methodBaseScale}, []any{&baseScaleRaw[i]})
		req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: target, Method: methodQuoteScale}, []any{&quoteScaleRaw[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, n)
	now := time.Now().Unix()
	for i, addr := range addrs {
		baseAddr := hexutil.Encode(baseHx[i][:])
		quoteAddr := hexutil.Encode(quoteHx[i][:])

		staticExtra, err := json.Marshal(StaticExtra{
			Base:       baseAddr,
			Quote:      quoteAddr,
			BaseScale:  baseScaleRaw[i].String(),
			QuoteScale: quoteScaleRaw[i].String(),
		})
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:     hexutil.Encode(addr[:]),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   now,
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      []*entity.PoolToken{{Address: baseAddr, Swappable: true}, {Address: quoteAddr, Swappable: true}},
			StaticExtra: string(staticExtra),
		})
	}
	return pools, nil
}
