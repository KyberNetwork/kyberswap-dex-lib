package composablestable

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
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
	)
	if err != nil {
		return p, err
	}

	// update pool

	reserves, err := t.initReserves(ctx, p.Tokens, rpcRes.PoolTokens)
	if err != nil {
		return p, err
	}

	extra, err := t.initExtra(ctx, rpcRes)
	if err != nil {
		return p, err
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

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

		feeTypes = []int{feeTypeSwap, feeTypeYield}
	)

	req := t.ethrpcClient.R().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: t.config.VaultAddress,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []interface{}{poolID},
	}, []interface{}{&poolTokens})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodTotalSupply,
	}, []interface{}{&bptTotalSupply})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetAmplificationParameter,
	}, []interface{}{&ampParams})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetLastJoinExitData,
	}, []interface{}{&lastJoinExit})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetRateProviders,
	}, []interface{}{&rateProviders})

	for i, token := range tokens {
		tokenAddr := common.HexToAddress(token.Address)

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetTokenRateCache,
			Params: []interface{}{tokenAddr},
		}, []interface{}{&tokenRateCaches[i]})

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodIsTokenExemptFromYieldProtocolFee,
			Params: []interface{}{tokenAddr},
		}, []interface{}{&isTokenExemptFromYieldProtocolFee[i]})
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetSwapFeePercentage,
	}, []interface{}{&swapFeePercentage})

	for _, feeType := range feeTypes {
		value := big.NewInt(0)
		protocolFeePercentageCache[feeType] = value

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetProtocolFeePercentageCache,
			Params: []interface{}{big.NewInt(int64(feeType))},
		}, []interface{}{&value})
	}

	if poolTypeVer >= poolTypeVer5 {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodIsExemptFromYieldProtocolFee,
		}, []interface{}{&isExemptFromYieldProtocolFee})
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodInRecoveryMode,
	}, []interface{}{&inRecoveryMode})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetPausedState,
	}, []interface{}{&pausedState})

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
	ctx context.Context,
	rpcRes *rpcRes,
) (*Extra, error) {
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

	extra := Extra{
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
