package composablestable

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

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

	// call RPC
	rpcRes, err := t.queryRPC(
		ctx,
		p.Address,
		common.HexToHash(staticExtra.PoolID),
		staticExtra.PoolTypeVer,
		p.Tokens,
		staticExtra.Vault,
		overrides,
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	// update pool

	reserves, err := t.initReserves(ctx, p.Tokens, rpcRes.PoolTokens)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	extra, err := t.initExtra(rpcRes, staticExtra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.BlockNumber = rpcRes.BlockNumber
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.BlockNumber = rpcRes.BlockNumber

	return p, nil
}

func (t *PoolTracker) queryRPC(
	ctx context.Context,
	poolAddress string,
	poolID common.Hash,
	poolTypeVer int,
	tokens []*entity.PoolToken,
	vault string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*rpcRes, error) {
	var (
		tokenNbr = len(tokens)

		poolTokens                        PoolTokensResp
		bptTotalSupply                    *big.Int
		ampParams                         AmplificationParameterResp
		lastJoinExit                      LastJoinExitResp
		rateProviders                     = make([]common.Address, tokenNbr)
		tokenRateCaches                   = make([]TokenRateCacheResp, tokenNbr)
		swapFeePercentage                 *big.Int
		protocolFeePercentageCache        = make(map[int]*big.Int)
		isTokenExemptFromYieldProtocolFee = make([]bool, tokenNbr)
		isExemptFromYieldProtocolFee      bool
		inRecoveryMode                    bool
		pausedState                       PausedStateResp

		blockNbr *big.Int

		feeTypes = []int{feeTypeSwap, feeTypeYield}
	)

	/*
		Call 1 get:
		- poolTokens
		- bptTotalSupply
		- ampParams
		- lastJoinExit
		- rateProviders
		- tokenRateCaches
		- swapFeePercentage
		- protocolFeePercentageCache
		- isTokenExemptFromYieldProtocolFee
		- isExemptFromYieldProtocolFee
		- inRecoveryMode
		- pausedState
	*/

	req := t.ethrpcClient.R().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []any{poolID},
	}, []any{&poolTokens})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodTotalSupply,
	}, []any{&bptTotalSupply})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetAmplificationParameter,
	}, []any{&ampParams})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetLastJoinExitData,
	}, []any{&lastJoinExit})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetRateProviders,
	}, []any{&rateProviders})

	for i, token := range tokens {
		if token.Swappable {
			tokenAddr := common.HexToAddress(token.Address)

			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: poolAddress,
				Method: poolMethodGetTokenRateCache,
				Params: []any{tokenAddr},
			}, []any{&tokenRateCaches[i]})

			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: poolAddress,
				Method: poolMethodIsTokenExemptFromYieldProtocolFee,
				Params: []any{tokenAddr},
			}, []any{&isTokenExemptFromYieldProtocolFee[i]})
		}
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetSwapFeePercentage,
	}, []any{&swapFeePercentage})

	for _, feeType := range feeTypes {
		value := big.NewInt(0)
		protocolFeePercentageCache[feeType] = value

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetProtocolFeePercentageCache,
			Params: []any{big.NewInt(int64(feeType))},
		}, []any{&value})
	}

	if poolTypeVer >= poolTypeVer5 {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodIsExemptFromYieldProtocolFee,
		}, []any{&isExemptFromYieldProtocolFee})
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodInRecoveryMode,
	}, []any{&inRecoveryMode})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetPausedState,
	}, []any{&pausedState})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, err
	}

	blockNbr = res.BlockNumber

	/*
		Update token rate
	*/

	canNotUpdateTokenRates := false
	req = t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(blockNbr)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	rateUpdatedTokenIndexes := make([]int, 0, len(tokens))
	updatedRate := make([]*big.Int, tokenNbr)
	for i, token := range tokens {
		if !token.Swappable || token.Address == poolAddress ||
			rateProviders[i].Hex() == zeroAddress || tokenRateCaches[i].Expires == nil ||
			time.Now().Unix() < tokenRateCaches[i].Expires.Int64() {
			continue
		}

		rateUpdatedTokenIndexes = append(rateUpdatedTokenIndexes, i)

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: rateProviders[i].Hex(),
			Method: poolMethodGetRate,
		}, []any{&updatedRate[i]})
	}

	if len(rateUpdatedTokenIndexes) > 0 {
		if _, err := req.Aggregate(); err != nil {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     DexType,
				"poolAddress": poolAddress,
			}).Warnf("can not update token rates: %s", err.Error())

			canNotUpdateTokenRates = true
		}

		for _, i := range rateUpdatedTokenIndexes {
			if updatedRate[i] == nil || tokenRateCaches[i].Duration == nil {
				continue
			}
			tokenRateCaches[i].Rate = updatedRate[i]
			tokenRateCaches[i].Expires = big.NewInt(time.Now().Unix() + tokenRateCaches[i].Duration.Int64())
		}
	}

	return &rpcRes{
		CanNotUpdateTokenRates:            canNotUpdateTokenRates,
		PoolTokens:                        poolTokens,
		BptTotalSupply:                    bptTotalSupply,
		Amp:                               ampParams.Value,
		LastJoinExit:                      lastJoinExit,
		RateProviders:                     rateProviders,
		TokenRateCaches:                   tokenRateCaches,
		SwapFeePercentage:                 swapFeePercentage,
		ProtocolFeePercentageCache:        protocolFeePercentageCache,
		IsTokenExemptFromYieldProtocolFee: isTokenExemptFromYieldProtocolFee,
		IsExemptFromYieldProtocolFee:      isExemptFromYieldProtocolFee,
		InRecoveryMode:                    inRecoveryMode,
		PausedState:                       pausedState,
		BlockNumber:                       res.BlockNumber.Uint64(),
	}, nil
}

func (t *PoolTracker) initExtra(
	rpcRes *rpcRes,
	staticExtra StaticExtra,
) (*Extra, error) {
	scalingFactors := make([]*uint256.Int, len(staticExtra.ScalingFactors))
	for i, scalingFactor := range staticExtra.ScalingFactors {
		var rate *uint256.Int
		if i == staticExtra.BptIndex || rpcRes.RateProviders[i].Hex() == zeroAddress {
			rate = number.Number_1e18
		} else {
			if rpcRes.TokenRateCaches[i].Rate != nil {
				rate, _ = uint256.FromBig(rpcRes.TokenRateCaches[i].Rate)
			}
		}

		scalingFactors[i] = scalingFactor

		if rate != nil {
			var err error
			scalingFactors[i], err = math.FixedPoint.MulDown(scalingFactor, rate)
			if err != nil {
				return nil, err
			}
		}
	}

	bptTotalSupply, overflow := uint256.FromBig(rpcRes.BptTotalSupply)
	if overflow {
		return nil, ErrOverflow
	}

	amp, overflow := uint256.FromBig(rpcRes.Amp)
	if overflow {
		return nil, ErrOverflow
	}

	var lastJoinExit LastJoinExitData
	lastJoinExit.LastJoinExitAmplification, _ = uint256.FromBig(
		rpcRes.LastJoinExit.LastJoinExitAmplification,
	)
	lastJoinExit.LastPostJoinExitInvariant, _ = uint256.FromBig(
		rpcRes.LastJoinExit.LastPostJoinExitInvariant,
	)

	rateProviders := make([]string, len(rpcRes.RateProviders))
	for i, rateProvider := range rpcRes.RateProviders {
		rateProviders[i] = strings.ToLower(rateProvider.Hex())
	}

	tokenRateCaches := make([]TokenRateCache, len(rpcRes.TokenRateCaches))
	for i, tokenRateCache := range rpcRes.TokenRateCaches {
		rate, _ := uint256.FromBig(tokenRateCache.Rate)
		oldRate, _ := uint256.FromBig(tokenRateCache.OldRate)
		duration, _ := uint256.FromBig(tokenRateCache.Duration)
		expires, _ := uint256.FromBig(tokenRateCache.Expires)
		tokenRateCaches[i] = TokenRateCache{
			Rate:     rate,
			OldRate:  oldRate,
			Duration: duration,
			Expires:  expires,
		}
	}

	swapFeePercentage, _ := uint256.FromBig(rpcRes.SwapFeePercentage)

	protocolFeePercentageCache := make(map[int]*uint256.Int)
	for feeType, value := range rpcRes.ProtocolFeePercentageCache {
		protocolFeePercentageCache[feeType], _ = uint256.FromBig(value)
	}

	isTokenExemptFromYieldProtocolFee := rpcRes.IsTokenExemptFromYieldProtocolFee

	isExemptFromYieldProtocolFee := rpcRes.IsExemptFromYieldProtocolFee

	inRecoveryMode := rpcRes.InRecoveryMode

	paused := !isNotPaused(rpcRes.PausedState)

	canNotUpdateTokenRates := rpcRes.CanNotUpdateTokenRates

	extra := Extra{
		CanNotUpdateTokenRates:            canNotUpdateTokenRates,
		ScalingFactors:                    scalingFactors,
		BptTotalSupply:                    bptTotalSupply,
		Amp:                               amp,
		LastJoinExit:                      lastJoinExit,
		RateProviders:                     rateProviders,
		TokenRateCaches:                   tokenRateCaches,
		SwapFeePercentage:                 swapFeePercentage,
		ProtocolFeePercentageCache:        protocolFeePercentageCache,
		IsTokenExemptFromYieldProtocolFee: isTokenExemptFromYieldProtocolFee,
		IsExemptFromYieldProtocolFee:      isExemptFromYieldProtocolFee,
		InRecoveryMode:                    inRecoveryMode,
		Paused:                            paused,
	}

	return &extra, nil
}

func (t *PoolTracker) initReserves(
	ctx context.Context,
	tokens []*entity.PoolToken,
	poolTokens PoolTokensResp,
) ([]string, error) {
	reserveByToken := make(map[string]*big.Int)
	for idx, token := range poolTokens.Tokens {
		addr := strings.ToLower(token.Hex())
		reserveByToken[addr] = poolTokens.Balances[idx]
	}

	reserves := make([]string, len(tokens))
	for idx, token := range tokens {
		r, ok := reserveByToken[token.Address]
		if !ok {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     DexType,
				"poolAddress": token.Address,
			}).Error("can not get reserve")

			return nil, ErrReserveNotFound
		}

		reserves[idx] = r.String()
	}

	return reserves, nil
}

func isNotPaused(pausedState PausedStateResp) bool {
	return time.Now().Unix() > pausedState.BufferPeriodEndTime.Int64() || !pausedState.Paused
}
