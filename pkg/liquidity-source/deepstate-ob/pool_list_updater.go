package deepstateob

import (
	"bytes"
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdater emits one entity.Pool per Config.Pairs entry. DeepstateV1
// pools are permissionless but there is no factory/creation event to scan,
// so this integration tracks a static, config-seeded set of pairs (today:
// just NVDA/USDG) instead of discovering them on-chain.
type PoolsListUpdater struct {
	cfg *Config
}

type poolsListUpdaterMetadata struct {
	Seeded bool `json:"seeded"`
}

var _ = poollist.RegisterFactoryC(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg}
}

func (u *PoolsListUpdater) GetNewPools(_ context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var meta poolsListUpdaterMetadata
	_ = json.Unmarshal(metadataBytes, &meta)
	if meta.Seeded {
		return nil, metadataBytes, nil
	}

	router := common.HexToAddress(u.cfg.Router)
	lens := hexutil.Encode(common.HexToAddress(u.cfg.Lens).Bytes())
	now := time.Now().Unix()
	pools := make([]entity.Pool, 0, len(u.cfg.Pairs))

	for _, pair := range u.cfg.Pairs {
		a, b := common.HexToAddress(pair.TokenA), common.HexToAddress(pair.TokenB)
		token0, token1 := a, b
		if bytes.Compare(a.Bytes(), b.Bytes()) > 0 {
			token0, token1 = b, a
		}

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			Router: hexutil.Encode(router.Bytes()),
			Lens:   lens,
		})

		pools = append(pools, entity.Pool{
			Address:   pairPoolAddress(router, token0, token1),
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(token0.Bytes()), Swappable: true},
				{Address: hexutil.Encode(token1.Bytes()), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	newMeta, _ := json.Marshal(poolsListUpdaterMetadata{Seeded: true})
	return pools, newMeta, nil
}

// pairPoolAddress synthesizes a stable, collision-free pool address for a
// singleton-router DEX with no per-pair pool contract. Follows the exact
// precedent set by pkg/liquidity-source/1010-prop's pairPoolAddress.
func pairPoolAddress(router, token0, token1 common.Address) string {
	hash := crypto.Keccak256(router.Bytes(), token0.Bytes(), token1.Bytes())
	return hexutil.Encode(common.BytesToAddress(hash).Bytes())
}
