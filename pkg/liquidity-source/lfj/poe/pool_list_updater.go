package poe

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	initialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.initialized {
		return nil, metadataBytes, nil
	}

	logger.Infof("started getting new pools")

	addresses := make([]common.Address, len(u.config.Pools))
	for i, p := range u.config.Pools {
		addresses[i] = common.HexToAddress(p)
	}

	pools, err := u.initPools(ctx, addresses)
	if err != nil {
		logger.Errorf("initPools failed: %v", err)
		return nil, metadataBytes, err
	}

	u.initialized = true

	logger.Infof("finished getting %d new pools", len(pools))

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, addresses []common.Address) ([]entity.Pool, error) {
	infos := make([]struct {
		tokenX common.Address
		tokenY common.Address
		oracle common.Address
	}, len(addresses))

	req := u.ethrpcClient.R().SetContext(ctx)
	for i, addr := range addresses {
		target := addr.Hex()
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: target,
			Method: "getTokenX",
		}, []any{&infos[i].tokenX}).AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: target,
			Method: "getTokenY",
		}, []any{&infos[i].tokenY}).AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: target,
			Method: "getOracle",
		}, []any{&infos[i].oracle})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(addresses))
	for i, addr := range addresses {
		info := infos[i]

		staticExtra, err := json.Marshal(StaticExtra{Oracle: info.oracle.Hex()})
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(addr[:]),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(info.tokenX[:]), Swappable: true},
				{Address: hexutil.Encode(info.tokenY[:]), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
		})
	}

	return pools, nil
}
