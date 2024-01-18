package gyroeclp

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/shared"
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

	rpcResp, err := t.queryRPC(ctx, p.Address, staticExtra.PoolID, staticExtra.Vault, p.Tokens, staticExtra.PoolTypeVer)
	if err != nil {
		return p, err
	}

	paused := !isNotPaused(rpcResp.PausedState)
	swapFeePercentage, _ := uint256.FromBig(rpcResp.SwapFeePercentage)
	paramsAlpha, _ := int256.FromBig(rpcResp.ECLPParamsResp.Params.Alpha)
	paramsBeta, _ := int256.FromBig(rpcResp.ECLPParamsResp.Params.Beta)
	paramsC, _ := int256.FromBig(rpcResp.ECLPParamsResp.Params.C)
	paramsS, _ := int256.FromBig(rpcResp.ECLPParamsResp.Params.S)
	paramsLambda, _ := int256.FromBig(rpcResp.ECLPParamsResp.Params.Lambda)
	tauAlphaX, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.TauAlpha.X)
	tauAlphaY, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.TauAlpha.Y)
	tauBetaX, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.TauBeta.X)
	tauBetaY, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.TauBeta.Y)
	u, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.U)
	v, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.V)
	w, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.W)
	z, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.Z)
	dSq, _ := int256.FromBig(rpcResp.ECLPParamsResp.D.DSq)

	var tokenRates []*uint256.Int
	if staticExtra.PoolTypeVer > poolTypeVer1 {
		tokenRates := make([]*uint256.Int, 2)
		tokenRates[0], _ = uint256.FromBig(rpcResp.TokenRatesResp.Rate0)
		tokenRates[1], _ = uint256.FromBig(rpcResp.TokenRatesResp.Rate1)
	}

	extra := Extra{
		Paused:            paused,
		SwapFeePercentage: swapFeePercentage,
		ParamsAlpha:       paramsAlpha,
		ParamsBeta:        paramsBeta,
		ParamsC:           paramsC,
		ParamsS:           paramsS,
		ParamsLambda:      paramsLambda,
		TauAlphaX:         tauAlphaX,
		TauAlphaY:         tauAlphaY,
		TauBetaX:          tauBetaX,
		TauBetaY:          tauBetaY,
		U:                 u,
		V:                 v,
		W:                 w,
		Z:                 z,
		DSq:               dSq,
		TokenRates:        tokenRates,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	reserves, err := t.initReserves(ctx, p, rpcResp.PoolTokens)
	if err != nil {
		return p, err
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.BlockNumber = rpcResp.BlockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) initReserves(
	ctx context.Context,
	p entity.Pool,
	poolTokens PoolTokensResp,
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
	tokens []*entity.PoolToken,
	poolTypeVer int,
) (*rpcResp, error) {
	var (
		poolTokens        PoolTokensResp
		swapFeePercentage *big.Int
		pausedState       PausedStateResp
		tokenRates        TokenRatesResp
		eclpParams        ECLPParamsResp

		poolIDHash = common.HexToHash(poolID)
	)

	req := t.ethrpcClient.R().
		SetContext(ctx).
		SetRequireSuccess(true)

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []interface{}{poolIDHash},
	}, []interface{}{&poolTokens})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetSwapFeePercentage,
	}, []interface{}{&swapFeePercentage})

	if poolTypeVer > poolTypeVer1 {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetTokenRates,
		}, []interface{}{&tokenRates})
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetECLPParams,
	}, []interface{}{&eclpParams})

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

	return &rpcResp{
		PoolTokens:        poolTokens,
		SwapFeePercentage: swapFeePercentage,
		PausedState:       pausedState,
		TokenRatesResp:    tokenRates,
		ECLPParamsResp:    eclpParams,
		BlockNumber:       res.BlockNumber.Uint64(),
	}, nil
}

func isNotPaused(pausedState PausedStateResp) bool {
	return time.Now().Unix() > pausedState.BufferPeriodEndTime.Int64() || !pausedState.Paused
}
