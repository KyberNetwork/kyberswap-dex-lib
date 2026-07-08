package prop

import (
	"context"
	"strings"
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

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

// poolsListUpdaterMetadata tracks the set of assets already known to the lister so that only
// new pairs (where at least one token is a newly discovered asset) are returned on each run.
type poolsListUpdaterMetadata struct {
	KnownAssets []string `json:"knownAssets"` // sorted lowercase hex addresses
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{"dexId": u.cfg.DexID})

	var meta poolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			log.Warnf("1010-prop: failed to parse metadata: %v", err)
		}
	}

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	var assets []common.Address
	req.AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: u.cfg.RouterAddress,
		Method: "getAssets",
		Params: nil,
	}, []any{&assets})

	if _, err := req.TryAggregate(); err != nil {
		log.Errorf("1010-prop: getAssets failed: %v", err)
		return nil, metadataBytes, err
	}

	// Build the set of previously known assets for O(1) lookup.
	knownSet := make(map[string]struct{}, len(meta.KnownAssets))
	for _, a := range meta.KnownAssets {
		knownSet[a] = struct{}{}
	}

	// Identify which assets are new so we know which pairs to emit.
	newAssetSet := make(map[string]struct{})
	allAddrs := make([]string, len(assets))
	for i, a := range assets {
		addr := hexutil.Encode(a[:])
		allAddrs[i] = addr
		if _, known := knownSet[addr]; !known {
			newAssetSet[addr] = struct{}{}
		}
	}

	if len(newAssetSet) == 0 {
		return nil, metadataBytes, nil
	}

	log.Infof("1010-prop: %d new asset(s) discovered, building pairs", len(newAssetSet))

	routerAddr := common.HexToAddress(u.cfg.RouterAddress)
	staticExtraBytes, _ := json.Marshal(StaticExtra{
		RouterAddress: strings.ToLower(u.cfg.RouterAddress),
	})

	now := time.Now().Unix()
	pools := make([]entity.Pool, 0, len(assets)*len(newAssetSet))

	for i := range assets {
		for j := i + 1; j < len(assets); j++ {
			// Only emit the pair if at least one token is newly discovered.
			_, iNew := newAssetSet[allAddrs[i]]
			_, jNew := newAssetSet[allAddrs[j]]
			if !iNew && !jNew {
				continue
			}
			poolAddr := pairPoolAddress(routerAddr, assets[i], assets[j])
			p := entity.Pool{
				Address:   poolAddr,
				Exchange:  u.cfg.DexID,
				Type:      DexType,
				Timestamp: now,
				Reserves:  entity.PoolReserves{"0", "0"},
				Tokens: []*entity.PoolToken{
					{Address: allAddrs[i], Swappable: true},
					{Address: allAddrs[j], Swappable: true},
				},
				Extra:       "{}",
				StaticExtra: string(staticExtraBytes),
			}
			pools = append(pools, p)
		}
	}

	newMeta, _ := json.Marshal(poolsListUpdaterMetadata{KnownAssets: allAddrs})
	return pools, newMeta, nil
}

// pairPoolAddress derives a deterministic pool address from the router and the two token addresses.
func pairPoolAddress(router, token0, token1 common.Address) string {
	hash := crypto.Keccak256(router.Bytes(), token0.Bytes(), token1.Bytes())
	return strings.ToLower(common.BytesToAddress(hash).Hex())
}
