package ilyris

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	cfg   *Config
	chain chainReader
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, chain: &ethrpcChain{client: client, lens: cfg.LensAddress}}
}

// GetNewPools pages through BinFactory.allPools; the cursor only advances on success.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var md Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &md); err != nil {
			return nil, metadataBytes, err
		}
	}
	limit := u.cfg.NewPoolLimit
	if limit <= 0 {
		limit = defaultNewPoolLimit
	}
	found, _, err := u.chain.FactoryPools(ctx, u.cfg.FactoryAddress, md.Offset, limit)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, len(found))
	for i, fp := range found {
		pools[i] = entity.Pool{
			Address:  fp.Address,
			Exchange: u.cfg.DexID,
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: fp.TokenX, Swappable: true},
				{Address: fp.TokenY, Swappable: true},
			},
		}
	}
	md.Offset += len(found)
	newMetadataBytes, err := json.Marshal(md)
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMetadataBytes, nil
}
