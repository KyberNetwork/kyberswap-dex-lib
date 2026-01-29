package smoothy

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
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

func (u *PoolsListUpdater) GetNewPools(_ context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	pool := entity.Pool{
		Address:   hexutil.Encode(u.config.Pool[:]),
		Exchange:  string(u.config.DexId),
		Type:      DexType,
		Extra:     "{}",
		Timestamp: time.Now().Unix(),
	}

	return []entity.Pool{pool}, nil, nil
}
