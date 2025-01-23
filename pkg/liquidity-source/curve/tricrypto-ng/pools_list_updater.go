package tricryptong

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config        shared.Config
	ethrpcClient  *ethrpc.Client
	sharedUpdater *shared.PoolsListUpdater
	logger        logger.Logger
}

func NewPoolsListUpdater(config *shared.Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": DexType,
	})

	sharedUpdater := shared.NewPoolsListUpdater(config, ethrpcClient, lg)

	return &PoolsListUpdater{
		config:        *config,
		ethrpcClient:  ethrpcClient,
		sharedUpdater: sharedUpdater,
		logger:        lg,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger.Infof("Start updating pools list ...")
	defer func() {
		u.logger.Infof("Finish updating pools list.")
	}()

	includedTypes := mapset.NewSet(shared.CURVE_POOL_TYPE_TRICRYPTO_NG)
	curvePools, newMetadataBytes, err := u.sharedUpdater.GetNewPools(ctx, metadataBytes, includedTypes)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(ctx, curvePools)
	if err != nil {
		u.logger.Error(err.Error())
		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) initPools(_ context.Context, curvePools []shared.CurvePoolWithType) ([]entity.Pool, error) {
	var pools = make([]entity.Pool, 0, len(curvePools))
	for _, curvePool := range curvePools {
		lg := u.logger.WithFields(logger.Fields{"poolAddress": curvePool.Address})

		if len(curvePool.LpTokenAddress) == 0 {
			lg.Warn("ignore pool with invalid LpTokenAddress")
			continue
		}

		if u.config.ChainID != valueobject.ChainIDSonic {
			if !SupportedImplementation.Contains(curvePool.Implementation) {
				lg.Debugf("ignore pool with implementation=%s", curvePool.Implementation)
				continue
			}
		}

		poolTokens := make([]*entity.PoolToken, 0, len(curvePool.Coins))
		reserves := make([]string, 0, len(curvePool.Coins))
		invalidDecimal := false
		isNativeCoins := make([]bool, 0, len(curvePool.Coins))
		for _, c := range curvePool.Coins {
			dec := c.GetDecimals()
			if dec == 0 {
				invalidDecimal = true
				break
			}
			poolTokens = append(poolTokens, &entity.PoolToken{
				Address:   strings.ToLower(c.Address),
				Symbol:    c.Symbol,
				Decimals:  dec,
				Swappable: true,
			})
			isNativeCoins = append(isNativeCoins, c.IsOrgNative)
			reserves = append(reserves, "0")
		}
		if invalidDecimal {
			lg.Warn("ignore pool with invalid coin decimal")
			continue
		}

		var staticExtra = StaticExtra{
			IsNativeCoins: isNativeCoins,
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			lg.Errorf("failed to marshal static extra data")
			return nil, err
		}

		newPool := entity.Pool{
			Address:     strings.ToLower(curvePool.Address),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      poolTokens,
			StaticExtra: string(staticExtraBytes),
		}
		pools = append(pools, newPool)
	}

	return pools, nil
}
