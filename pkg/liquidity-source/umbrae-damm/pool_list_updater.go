package umbraedamm

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	initialized  bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools discovers the configured DAMM pairs once. DAMM has no factory, so the pair set is
// static and supplied via config. Reserves are left at 0 — the tracker fills them in.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.initialized {
		return nil, metadataBytes, nil
	}

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Info("started getting new pools")

	pools, err := u.initPools(ctx, u.config.Pools)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": u.config.DexID, "err": err}).Error("initPools failed")
		return nil, metadataBytes, err
	}

	u.initialized = true
	logger.WithFields(logger.Fields{"dex_id": u.config.DexID, "pools_len": len(pools)}).Info("finished getting new pools")

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, addresses []string) ([]entity.Pool, error) {
	tokenX := make([]common.Address, len(addresses))
	tokenY := make([]common.Address, len(addresses))

	req := u.ethrpcClient.R().SetContext(ctx)
	for i, addr := range addresses {
		req.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: addr,
			Method: pairMethodTokenX,
		}, []any{&tokenX[i]}).AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: addr,
			Method: pairMethodTokenY,
		}, []any{&tokenY[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(addresses))
	for i, addr := range addresses {
		pools = append(pools, entity.Pool{
			Address:   strings.ToLower(addr),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(tokenX[i].Hex()), Swappable: true},
				{Address: strings.ToLower(tokenY[i].Hex()), Swappable: true},
			},
			Extra: "{}",
		})
	}

	return pools, nil
}
