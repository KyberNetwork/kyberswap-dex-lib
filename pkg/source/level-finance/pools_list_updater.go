package levelfinance

import (
	"context"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"time"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:         cfg,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		return nil, nil, nil
	}

	pools, err := d.init(ctx)
	if err != nil {
		return nil, nil, err
	}

	d.hasInitialized = true

	return pools, nil, nil
}

func (d *PoolsListUpdater) init(ctx context.Context) ([]entity.Pool, error) {
	var (
		allAssets = make([]common.Address, 10)
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < 10; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    LiquidityPoolAbi,
			Target: d.config.LiquidityPoolAddress,
			Method: liquidityPoolMethodAllAssets,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&allAssets[i]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.Errorf("failed to aggregate call with error %v", err)
		return nil, err
	}

	reserves := make([]string, 0)
	tokens := make([]*entity.PoolToken, 0)

	for _, tokenAddress := range allAssets {
		if tokenAddress.Hex() == valueobject.ZeroAddress {
			break
		}
		tokens = append(tokens, &entity.PoolToken{
			Address:   strings.ToLower(tokenAddress.Hex()),
			Weight:    defaultWeight,
			Swappable: true,
		})
		reserves = append(reserves, zeroString)
	}

	var newPool = entity.Pool{
		Address:   strings.ToLower(d.config.LiquidityPoolAddress),
		Exchange:  d.config.DexID,
		Type:      DexTypeLevelFinance,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
	}

	logger.Infof("[%s] got pool %v from config", d.config.DexID, newPool.Address)

	return []entity.Pool{newPool}, nil
}
