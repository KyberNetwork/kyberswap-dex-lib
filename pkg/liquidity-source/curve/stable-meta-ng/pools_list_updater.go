package stablemetang

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/bytedance/sonic"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
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

	includedTypes := mapset.NewSet(shared.CURVE_POOL_TYPE_STABLE_NG_META)
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

func (u *PoolsListUpdater) initPools(ctx context.Context, curvePools []shared.CurvePoolWithType) ([]entity.Pool, error) {
	var (
		aList          = make([]*big.Int, len(curvePools))
		aPreciseList   = make([]*big.Int, len(curvePools))
		feeMultipliers = make([]*big.Int, len(curvePools))
	)

	calls := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, curvePool := range curvePools {
		calls.AddCall(&ethrpc.Call{
			ABI:    curveStableMetaNGABI,
			Target: curvePool.Address,
			Method: poolMethodA,
			Params: nil,
		}, []interface{}{&aList[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    curveStableMetaNGABI,
			Target: curvePool.Address,
			Method: poolMethodAPrecise,
			Params: nil,
		}, []interface{}{&aPreciseList[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    curveStableMetaNGABI,
			Target: curvePool.Address,
			Method: poolMethodOffpegFeeMul,
			Params: nil,
		}, []interface{}{&feeMultipliers[i]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		u.logger.Errorf("failed to aggregate call to get pool data %v", err)
		return nil, err
	}

	var pools = make([]entity.Pool, 0, len(curvePools))
	for i, curvePool := range curvePools {
		lg := u.logger.WithFields(logger.Fields{"poolAddress": curvePool.Address})

		if len(curvePool.LpTokenAddress) == 0 {
			lg.Warn("ignore pool with invalid LpTokenAddress")
			continue
		}

		poolTokens := make([]*entity.PoolToken, 0, len(curvePool.Coins))
		reserves := make([]string, 0, len(curvePool.Coins)+1) // N coins & totalSupply
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
		reserves = append(reserves, "0")

		var staticExtra = StaticExtra{
			IsNativeCoins:    isNativeCoins,
			BasePool:         strings.ToLower(curvePool.BasePoolAddress),
			UnderlyingTokens: lo.Map(curvePool.UnderlyingCoins, func(c shared.CurveCoin, _ int) string { return strings.ToLower(c.Address) }),
		}

		if aList[i] != nil && aPreciseList[i] != nil {
			staticExtra.APrecision = new(uint256.Int).Div(number.SetFromBig(aPreciseList[i]), number.SetFromBig(aList[i]))
		} else if aList[i] != nil {
			staticExtra.APrecision = uint256.NewInt(1)
		} else {
			lg.Warn("ignore pool with unknown APrecision")
			continue
		}

		staticExtra.OffpegFeeMultiplier = uint256.MustFromBig(feeMultipliers[i])

		staticExtraBytes, err := sonic.Marshal(staticExtra)
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
