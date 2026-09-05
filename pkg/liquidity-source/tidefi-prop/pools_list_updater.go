package tidefiprop

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdater builds pools from a fixed, config-provided token list:
// TideFi has no on-chain enumerable asset registry (pricing config is
// pushed by a trusted off-chain signer with no corresponding event log), so
// unlike 1010-prop's getAssets()-driven discovery, the supported token set
// has to come from config until/unless an off-chain source of truth is
// wired in. Every configured token is assumed tradeable against every
// other; the tracker's sampled quotes naturally return an empty ladder for
// any pair TideFi doesn't actually support.
type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(_ context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if metadataBytes != nil {
		return nil, metadataBytes, nil
	}

	staticExtraBytes, _ := json.Marshal(StaticExtra{
		Address: strings.ToLower(u.cfg.Address),
	})

	now := time.Now().Unix()
	tokens := u.cfg.Tokens
	pools := make([]entity.Pool, 0, len(tokens)*(len(tokens)-1)/2)

	for i := range tokens {
		for j := i + 1; j < len(tokens); j++ {
			pools = append(pools, entity.Pool{
				Address:   poolAddress(tokens[i], tokens[j]),
				Exchange:  u.cfg.DexID,
				Type:      DexType,
				Timestamp: now,
				Reserves:  entity.PoolReserves{"0", "0"},
				Tokens: []*entity.PoolToken{
					{Address: strings.ToLower(tokens[i]), Swappable: true},
					{Address: strings.ToLower(tokens[j]), Swappable: true},
				},
				Extra:       "{}",
				StaticExtra: string(staticExtraBytes),
			})
		}
	}

	return pools, []byte("done"), nil
}

// poolAddress is synthetic, following manta-prop's pattern: TideFi has no
// per-pair contract of its own (it's a single fixed swapper backing every
// pair), so a fixed namespace plus the pair already disambiguates every
// pool without needing the swapper address itself.
func poolAddress(token0, token1 string) string {
	return "tidefiprop_" + strings.ToLower(token0) + "_" + strings.ToLower(token1)
}
