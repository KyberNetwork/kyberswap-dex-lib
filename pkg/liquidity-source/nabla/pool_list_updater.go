package nabla

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	pools := make([]entity.Pool, 0)
	var routers []common.Address
	if _, err := u.ethrpcClient.R().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    portalABI,
			Target: u.config.Portal,
			Method: "getRouters",
		}, []any{&routers}).
		Call(); err != nil {
		logger.Errorf("failed to get routers")
		return nil, nil, err
	}

	for _, router := range routers {
		var oracle common.Address
		if _, err := u.ethrpcClient.R().
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    portalABI,
				Target: u.config.Portal,
				Method: "oracleAdapter",
			}, []any{&oracle}).
			Call(); err != nil {
			logger.Errorf("failed to get oracle")
			return nil, nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(router[:]),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Extra:     "{}",
		})
	}

	return pools, nil, nil
}
