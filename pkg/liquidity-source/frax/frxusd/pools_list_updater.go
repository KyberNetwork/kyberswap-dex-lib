package frxusd

import (
	"context"
	"strings"
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
	cfg          *Config
	ethrpcClient *ethrpc.Client

	initialized bool
}

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.Infof("start updating pools list")
	defer func() {
		logger.Infof("finish updating pools list")
	}()

	if u.initialized {
		return nil, metadataBytes, nil
	}

	pools := make([]entity.Pool, 0, len(u.cfg.Vaults))
	for p := range u.cfg.Vaults {
		pool, err := u.getNewPool(ctx, p)
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, *pool)
	}

	u.initialized = true

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) getNewPool(ctx context.Context, p string) (*entity.Pool, error) {
	var (
		asset, share common.Address
	)
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI: FrxUsdCustodianUsdcABI, Target: p, Method: "frxUSD",
		}, []any{&share}).
		AddCall(&ethrpc.Call{
			ABI: FrxUsdCustodianUsdcABI, Target: p, Method: "asset",
		}, []any{&asset}).Aggregate(); err != nil {
		logger.Errorf("aggregate state failed err %v", err)
		return nil, err
	}

	return &entity.Pool{
		Address:   strings.ToLower(p),
		Exchange:  u.cfg.DexId,
		Type:      DexType,
		Reserves:  []string{"0", "0"},
		Timestamp: time.Now().Unix(),
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(share[:]), Swappable: true},
			{Address: hexutil.Encode(asset[:]), Swappable: true},
		},
		Extra: "{}",
	}, nil
}
