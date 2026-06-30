package backup

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	gyroeclp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/eclp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *gyroeclp.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterBackupFactoryCE(gyroeclp.DexType, NewPoolTracker)

func NewPoolTracker(
	config *gyroeclp.Config,
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
		"dexType":     gyroeclp.DexType,
		"poolAddress": p.Address,
	}).Info("Start updating state ...")

	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     gyroeclp.DexType,
			"poolAddress": p.Address,
		}).Info("Finish updating state.")
	}()

	var staticExtra gyroeclp.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     gyroeclp.DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	rpcResp, err := t.queryRPC(ctx, p.Address, staticExtra.PoolID, staticExtra.Vault, p.Tokens, staticExtra.PoolTypeVer, overrides)
	if err != nil {
		return p, err
	}

	paused := !gyroeclp.IsNotPaused(rpcResp.PausedState)
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
	if staticExtra.PoolTypeVer > gyroeclp.PoolTypeVer1 {
		tokenRates = make([]*uint256.Int, 2)
		tokenRates[0], _ = uint256.FromBig(rpcResp.TokenRatesResp.Rate0)
		tokenRates[1], _ = uint256.FromBig(rpcResp.TokenRatesResp.Rate1)
	}

	extra := gyroeclp.Extra{
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

	reserves, err := t.initReserves(p, rpcResp.PoolTokens)
	if err != nil {
		return p, err
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.BlockNumber = rpcResp.BlockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) initReserves(p entity.Pool, poolTokens gyroeclp.PoolTokensResp) ([]string, error) {
	reserveByToken := make(map[string]*big.Int)
	for idx, token := range poolTokens.Tokens {
		addr := hexutil.Encode(token[:])
		reserveByToken[addr] = poolTokens.Balances[idx]
	}

	reserves := make([]string, len(p.Tokens))
	for idx, token := range p.Tokens {
		r, ok := reserveByToken[token.Address]
		if !ok {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     gyroeclp.DexType,
				"poolAddress": p.Address,
			}).Error("can not get reserve")
			return nil, gyroeclp.ErrReserveNotFound
		}
		reserves[idx] = r.String()
	}

	return reserves, nil
}

func (t *PoolTracker) queryRPC(
	ctx context.Context,
	poolAddress, poolID, vault string,
	tokens []*entity.PoolToken,
	poolTypeVer int,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*gyroeclp.RPCResp, error) {
	d := &gyroeclp.RPCResp{}
	poolIDHash := common.HexToHash(poolID)

	req := t.ethrpcClient.R().SetContext(ctx).SetRequireSuccess(true).SetOverrides(overrides)

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []any{poolIDHash},
	}, []any{&d.PoolTokens})

	req.AddCall(&ethrpc.Call{
		ABI:    *gyroeclp.PoolABI,
		Target: poolAddress,
		Method: gyroeclp.PoolMethodGetSwapFeePercentage,
	}, []any{&d.SwapFeePercentage})

	if poolTypeVer > gyroeclp.PoolTypeVer1 {
		req.AddCall(&ethrpc.Call{
			ABI:    *gyroeclp.PoolABI,
			Target: poolAddress,
			Method: gyroeclp.PoolMethodGetTokenRates,
		}, []any{&d.TokenRatesResp})
	}

	req.AddCall(&ethrpc.Call{
		ABI:    *gyroeclp.PoolABI,
		Target: poolAddress,
		Method: gyroeclp.PoolMethodGetECLPParams,
	}, []any{&d.ECLPParamsResp})

	req.AddCall(&ethrpc.Call{
		ABI:    *gyroeclp.PoolABI,
		Target: poolAddress,
		Method: gyroeclp.PoolMethodGetPausedState,
	}, []any{&d.PausedState})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     gyroeclp.DexType,
			"poolAddress": poolAddress,
		}).Error(err.Error())
		return nil, err
	}

	d.BlockNumber = res.BlockNumber.Uint64()
	return d, nil
}
