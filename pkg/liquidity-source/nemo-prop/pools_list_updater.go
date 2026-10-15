package nemoprop

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdater discovers one pool per (base, market) from the proxy's
// getMarkets_v1. The market list belongs to the pricing implementation, so
// a proxy upgrade can add, drop or reorder markets: pools are keyed by
// asset address (never by position), and a dropped market's pool simply
// stops quoting once the feed no longer covers it.
type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

// Metadata records every pool address already emitted, so later runs only
// return markets added by a pricing upgrade.
type Metadata struct {
	Seen map[string]bool `json:"seen"`
}

type marketsResult struct {
	Base    common.Address
	Markets []byte
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

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

	var result marketsResult
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    nemoSwapABI,
		Target: u.cfg.Address,
		Method: "getMarkets_v1",
	}, []any{&result}).Call(); err != nil {
		logger.WithFields(logger.Fields{"dexId": u.cfg.DexID}).Errorf("nemo-prop: getMarkets_v1 failed: %v", err)
		return nil, metadataBytes, err
	}

	assets, err := decodeMarkets(result.Base, result.Markets)
	if err != nil {
		return nil, metadataBytes, err
	}

	proxy := common.HexToAddress(u.cfg.Address)
	staticExtraBytes, err := json.Marshal(StaticExtra{Address: hexutil.Encode(proxy[:])})
	if err != nil {
		return nil, metadataBytes, err
	}

	now := time.Now().Unix()
	pools := make([]entity.Pool, 0, len(assets))
	for _, asset := range assets {
		addr := poolAddress(proxy, result.Base, asset)
		if metadata.Seen[addr] {
			continue
		}
		metadata.Seen[addr] = true

		pools = append(pools, entity.Pool{
			Address:   addr,
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(result.Base[:]), Swappable: true},
				{Address: hexutil.Encode(asset[:]), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMetadataBytes, nil
}

// decodeMarkets splits getMarkets_v1's packed entries,
// [asset: 160 bits][unit: 96 bits] each, into asset addresses. Units are
// unused.
func decodeMarkets(base common.Address, markets []byte) ([]common.Address, error) {
	if len(markets)%marketEntrySize != 0 || base == (common.Address{}) {
		return nil, ErrInvalidMarkets
	}
	assets := make([]common.Address, 0, len(markets)/marketEntrySize)
	for i := 0; i < len(markets); i += marketEntrySize {
		asset := common.BytesToAddress(markets[i : i+common.AddressLength])
		if asset == (common.Address{}) || asset == base {
			return nil, ErrInvalidMarkets
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

// poolAddress is synthetic and deterministic: one pool per
// (proxy, base, market).
func poolAddress(proxy, base, market common.Address) string {
	addr := common.BytesToAddress(crypto.Keccak256(proxy[:], base[:], market[:]))
	return hexutil.Encode(addr[:])
}
