package angletransmuter

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
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

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	var collateralList []common.Address
	var agToken common.Address
	if _, err := calls.AddCall(&ethrpc.Call{
		ABI:    transmuterABI,
		Target: d.config.Transmuter,
		Method: "getCollateralList",
	}, []any{&collateralList}).AddCall(&ethrpc.Call{
		ABI:    transmuterABI,
		Target: d.config.Transmuter,
		Method: "agToken",
	}, []any{&agToken}).Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, nil, err
	}

	tokens := append(collateralList, agToken)

	pools := []entity.Pool{
		{
			Address:   d.config.Transmuter,
			Exchange:  d.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: lo.Map(tokens, func(token common.Address, _ int) *entity.PoolToken {
				return &entity.PoolToken{
					Address:   strings.ToLower(token.Hex()),
					Swappable: true,
				}
			}),
			Reserves: lo.Map(tokens, func(token common.Address, _ int) string {
				return "0"
			}),
		},
	}

	logger.WithFields(logger.Fields{"pool": pools}).Info("finish fetching pools")

	d.hasInitialized = true
	return pools, nil, nil
}
