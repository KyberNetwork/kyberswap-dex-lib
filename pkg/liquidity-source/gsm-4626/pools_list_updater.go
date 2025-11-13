package gsm4626

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	logger      logger.Logger
	initialized bool
}

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.Infof("start updating pools list")
	defer func() {
		logger.Infof("finish updating pools list")
	}()

	if u.initialized {
		return nil, metadataBytes, nil
	}

	pools := make([]entity.Pool, 0, len(u.cfg.GSMs))
	for _, gsm := range u.cfg.GSMs {
		pool, err := u.getNewPool(ctx, gsm)
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, *pool)
	}

	u.initialized = true

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) getNewPool(ctx context.Context, gsm string) (*entity.Pool, error) {
	var (
		priceStrategy   common.Address
		ghoToken        common.Address
		underlyingAsset common.Address
		priceRatio      *big.Int
	)
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI: gsm4626ABI, Target: gsm, Method: gsmMethodPriceStrategy,
		}, []any{&priceStrategy}).
		AddCall(&ethrpc.Call{
			ABI: gsm4626ABI, Target: gsm, Method: gsmMethodGhoToken,
		}, []any{&ghoToken}).
		AddCall(&ethrpc.Call{
			ABI: gsm4626ABI, Target: gsm, Method: gsmMethodUnderlyingAsset,
		}, []any{&underlyingAsset}).Aggregate(); err != nil {
		logger.Errorf("aggregate state failed err %v", err)
		return nil, err
	}

	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI: priceStrategyABI, Target: priceStrategy.String(), Method: priceStrategyMethodPriceRatio,
		}, []any{&priceRatio}).Call(); err != nil {
		logger.Errorf("get price ratio faied err %v", err)
		return nil, err
	}

	extraBytes, err := json.Marshal(StaticExtra{PriceRatio: uint256.MustFromBig(priceRatio)})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:  strings.ToLower(gsm),
		Exchange: u.cfg.DexId,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(ghoToken[:]), Swappable: true},
			{Address: hexutil.Encode(underlyingAsset[:]), Swappable: true},
		},
		StaticExtra: string(extraBytes),
	}, nil
}
