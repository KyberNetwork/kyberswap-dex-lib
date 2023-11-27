package weighted

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/vault"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexId":       t.config.DexID,
		"dexType":     DexTypeBalancerWeighted,
		"poolAddress": p.Address,
	}).Info("Start updating state ...")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexTypeBalancerWeighted,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	var (
		poolTokens        PoolTokens
		swapFeePercentage *big.Int
		scalingFactors    []*big.Int
	)

	req := t.ethrpcClient.R().
		SetContext(ctx).
		SetRequireSuccess(true)

	req.AddCall(&ethrpc.Call{
		ABI:    vault.ABI,
		Target: t.config.VaultAddress,
		Method: vault.MethodGetPoolTokens,
		Params: []interface{}{common.HexToHash(staticExtra.PoolID)},
	}, []interface{}{&poolTokens})

	req.AddCall(&ethrpc.Call{
		ABI:    weightedPoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFeePercentage,
	}, []interface{}{&swapFeePercentage})

	req.AddCall(&ethrpc.Call{
		ABI:    weightedPoolABI,
		Target: p.Address,
		Method: poolMethodGetScalingFactors,
	}, []interface{}{&scalingFactors})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexTypeBalancerWeighted,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	extra := Extra{
		SwapFeePercentage: swapFeePercentage,
		ScalingFactors:    scalingFactors,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexTypeBalancerWeighted,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	p.BlockNumber = res.BlockNumber.Uint64()
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
