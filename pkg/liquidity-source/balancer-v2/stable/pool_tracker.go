package stable

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var ErrReserveNotFound = errors.New("reserve not found")

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
		"dexType":     DexType,
		"poolAddress": p.Address,
	}).Info("Start updating state ...")

	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Info("Finish updating state.")
	}()

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	var oldExtra Extra
	if err := json.Unmarshal([]byte(p.Extra), &oldExtra); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}
	scalingFactors := oldExtra.ScalingFactors

	// call RPC
	rpcRes, err := t.queryRPC(ctx, p.Address, staticExtra.PoolID, staticExtra.Vault, staticExtra.PoolType)
	if err != nil {
		return p, err
	}

	var (
		amp, _               = uint256.FromBig(rpcRes.Amp)
		swapFeePercentage, _ = uint256.FromBig(rpcRes.SwapFeePercentage)
		poolTokens           = rpcRes.PoolTokens
		pausedState          = rpcRes.PausedState
		blockNumber          = rpcRes.BlockNumber
	)

	if staticExtra.PoolType == poolTypeMetaStable {
		factors := make([]*uint256.Int, len(rpcRes.ScalingFactors))
		for idx, factor := range rpcRes.ScalingFactors {
			factors[idx], _ = uint256.FromBig(factor)
		}

		scalingFactors = factors
	}

	// update pool

	extra := Extra{
		Amp:               amp,
		SwapFeePercentage: swapFeePercentage,
		ScalingFactors:    scalingFactors,
		Paused:            !isNotPaused(pausedState),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	reserves, err := t.initReserves(ctx, p, poolTokens)
	if err != nil {
		return p, err
	}

	p.BlockNumber = blockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}

func (t *PoolTracker) initReserves(
	ctx context.Context,
	p entity.Pool,
	poolTokens PoolTokens,
) ([]string, error) {
	reserveByToken := make(map[string]*big.Int)
	for idx, token := range poolTokens.Tokens {
		addr := strings.ToLower(token.Hex())
		reserveByToken[addr] = poolTokens.Balances[idx]
	}

	reserves := make([]string, len(p.Tokens))
	for idx, token := range p.Tokens {
		r, ok := reserveByToken[token.Address]
		if !ok {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     DexType,
				"poolAddress": p.Address,
			}).Error("can not get reserve")

			return nil, ErrReserveNotFound
		}

		reserves[idx] = r.String()
	}

	return reserves, nil
}

func (t *PoolTracker) queryRPC(
	ctx context.Context,
	poolAddress string,
	poolID string,
	vault string,
	poolType string,
) (*rpcRes, error) {
	var (
		poolTokens        PoolTokens
		swapFeePercentage *big.Int
		pausedState       PausedState
		ampParams         AmplificationParameter
		scalingFactors    []*big.Int
	)

	req := t.ethrpcClient.R().
		SetContext(ctx).
		SetRequireSuccess(true)

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []interface{}{common.HexToHash(poolID)},
	}, []interface{}{&poolTokens})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetAmplificationParameter,
	}, []interface{}{&ampParams})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetSwapFeePercentage,
	}, []interface{}{&swapFeePercentage})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetPausedState,
	}, []interface{}{&pausedState})

	if poolType == poolTypeMetaStable {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetScalingFactors,
		}, []interface{}{&scalingFactors})
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": poolAddress,
		}).Error(err.Error())

		return nil, err
	}

	return &rpcRes{
		Amp:               ampParams.Value,
		PoolTokens:        poolTokens,
		SwapFeePercentage: swapFeePercentage,
		ScalingFactors:    scalingFactors,
		PausedState:       pausedState,
		BlockNumber:       res.BlockNumber.Uint64(),
	}, nil
}

func isNotPaused(pausedState PausedState) bool {
	return time.Now().Unix() > pausedState.BufferPeriodEndTime.Int64() || !pausedState.Paused
}
