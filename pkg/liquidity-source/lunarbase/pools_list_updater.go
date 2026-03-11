package lunarbase

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	if config.DexID == "" {
		config.DexID = DexType
	}
	if config.ChainID == 0 {
		config.ChainID = valueobject.ChainIDBase
	}

	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, metadataBytes, nil
	}

	state, err := fetchRPCState(ctx, u.config, u.ethrpcClient, nil)
	if err != nil {
		return nil, metadataBytes, err
	}

	poolEntity, err := buildEntityPool(u.config, state)
	if err != nil {
		return nil, metadataBytes, err
	}
	poolEntity.Timestamp = time.Now().Unix()

	u.hasInitialized = true

	metadataBytes, err = json.Marshal(Metadata{Initialized: true})
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{poolEntity}, metadataBytes, nil
}
