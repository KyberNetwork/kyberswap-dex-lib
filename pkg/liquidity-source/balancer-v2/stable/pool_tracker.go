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
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var ErrReserveNotFound = errors.New("reserve not found")

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *shared.Config,
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
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
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
	rpcRes, err := t.queryRPC(ctx, p.Address, staticExtra.PoolID, staticExtra.Vault, staticExtra.PoolType, overrides)
	if err != nil {
		return p, err
	}

	var (
		amp, _                       = uint256.FromBig(rpcRes.Amp)
		swapFeePercentage, _         = uint256.FromBig(rpcRes.SwapFeePercentage)
		protocolSwapFeePercentage, _ = uint256.FromBig(rpcRes.ProtocolSwapFeePercentage)
		poolTokens                   = rpcRes.PoolTokens
		pausedState                  = rpcRes.PausedState
		blockNumber                  = rpcRes.BlockNumber
	)

	if staticExtra.PoolType == poolTypeMetaStable || staticExtra.PoolType == poolTypeLegacyMetaStable {
		factors := make([]*uint256.Int, len(rpcRes.ScalingFactors))
		for idx, factor := range rpcRes.ScalingFactors {
			factors[idx], _ = uint256.FromBig(factor)
		}

		scalingFactors = factors
	}

	// update pool

	extra := Extra{
		Amp:                       amp,
		SwapFeePercentage:         swapFeePercentage,
		ProtocolSwapFeePercentage: protocolSwapFeePercentage,
		ScalingFactors:            scalingFactors,
		Paused:                    !isNotPaused(pausedState),
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

	reserves, err := t.initReserves(p, poolTokens)
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
	overrides map[common.Address]gethclient.OverrideAccount,
) (*rpcRes, error) {
	var (
		poolTokens                                   PoolTokens
		protocolSwapFeePercentage, swapFeePercentage *big.Int
		pausedState                                  PausedState
		ampParams                                    AmplificationParameter
		scalingFactors                               []*big.Int
	)

	req := t.ethrpcClient.R().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []any{common.HexToHash(poolID)},
	}, []any{&poolTokens})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetAmplificationParameter,
	}, []any{&ampParams})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetSwapFeePercentage,
	}, []any{&swapFeePercentage})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetPausedState,
	}, []any{&pausedState})

	if poolType == poolTypeMetaStable || poolType == poolTypeLegacyMetaStable {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetScalingFactors,
		}, []any{&scalingFactors})
	}

	if t.config.ProtocolFeesCollector != "" {
		req.AddCall(&ethrpc.Call{
			ABI:    shared.ProtocolFeesCollectorABI,
			Target: t.config.ProtocolFeesCollector,
			Method: protocolMethodGetSwapFeePercentage,
		}, []any{&protocolSwapFeePercentage})
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
		Amp:                       ampParams.Value,
		PoolTokens:                poolTokens,
		SwapFeePercentage:         swapFeePercentage,
		ProtocolSwapFeePercentage: protocolSwapFeePercentage,
		ScalingFactors:            scalingFactors,
		PausedState:               pausedState,
		BlockNumber:               res.BlockNumber.Uint64(),
	}, nil
}

func isNotPaused(pausedState PausedState) bool {
	return time.Now().Unix() > pausedState.BufferPeriodEndTime.Int64() || !pausedState.Paused
}
