package lazy

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

var _ = pooltrack.RegisterFactoryCE(gyroeclp.DexType, NewPoolTracker)

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
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
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

	d := &gyroeclp.RPCResp{}
	req := t.ethrpcClient.R().SetContext(ctx).SetRequireSuccess(true).SetOverrides(overrides)
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, staticExtra.Vault, staticExtra.PoolID, staticExtra.PoolTypeVer, d)

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     gyroeclp.DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	d.BlockNumber = res.BlockNumber.Uint64()
	return buildPoolState(p, &staticExtra, d)
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress, vault, poolID string, poolTypeVer int, d *gyroeclp.RPCResp) {
	poolIDHash := common.HexToHash(poolID)
	addFn(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []any{poolIDHash},
	}, []any{&d.PoolTokens})
	addFn(&ethrpc.Call{
		ABI:    gyroeclp.PoolABI,
		Target: poolAddress,
		Method: gyroeclp.PoolMethodGetSwapFeePercentage,
	}, []any{&d.SwapFeePercentage})
	if poolTypeVer > gyroeclp.PoolTypeVer1 {
		addFn(&ethrpc.Call{
			ABI:    gyroeclp.PoolABI,
			Target: poolAddress,
			Method: gyroeclp.PoolMethodGetTokenRates,
		}, []any{&d.TokenRatesResp})
	}
	addFn(&ethrpc.Call{
		ABI:    gyroeclp.PoolABI,
		Target: poolAddress,
		Method: gyroeclp.PoolMethodGetECLPParams,
	}, []any{&d.ECLPParamsResp})
	addFn(&ethrpc.Call{
		ABI:    gyroeclp.PoolABI,
		Target: poolAddress,
		Method: gyroeclp.PoolMethodGetPausedState,
	}, []any{&d.PausedState})
}

func buildPoolState(p entity.Pool, staticExtra *gyroeclp.StaticExtra, d *gyroeclp.RPCResp) (entity.Pool, error) {
	paused := !gyroeclp.IsNotPaused(d.PausedState)
	swapFeePercentage, _ := uint256.FromBig(d.SwapFeePercentage)
	paramsAlpha, _ := int256.FromBig(d.ECLPParamsResp.Params.Alpha)
	paramsBeta, _ := int256.FromBig(d.ECLPParamsResp.Params.Beta)
	paramsC, _ := int256.FromBig(d.ECLPParamsResp.Params.C)
	paramsS, _ := int256.FromBig(d.ECLPParamsResp.Params.S)
	paramsLambda, _ := int256.FromBig(d.ECLPParamsResp.Params.Lambda)
	tauAlphaX, _ := int256.FromBig(d.ECLPParamsResp.D.TauAlpha.X)
	tauAlphaY, _ := int256.FromBig(d.ECLPParamsResp.D.TauAlpha.Y)
	tauBetaX, _ := int256.FromBig(d.ECLPParamsResp.D.TauBeta.X)
	tauBetaY, _ := int256.FromBig(d.ECLPParamsResp.D.TauBeta.Y)
	u, _ := int256.FromBig(d.ECLPParamsResp.D.U)
	v, _ := int256.FromBig(d.ECLPParamsResp.D.V)
	w, _ := int256.FromBig(d.ECLPParamsResp.D.W)
	z, _ := int256.FromBig(d.ECLPParamsResp.D.Z)
	dSq, _ := int256.FromBig(d.ECLPParamsResp.D.DSq)

	var tokenRates []*uint256.Int
	if staticExtra.PoolTypeVer > gyroeclp.PoolTypeVer1 {
		tokenRates = make([]*uint256.Int, 2)
		tokenRates[0], _ = uint256.FromBig(d.TokenRatesResp.Rate0)
		tokenRates[1], _ = uint256.FromBig(d.TokenRatesResp.Rate1)
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

	reserves, err := initReserves(p, d.PoolTokens)
	if err != nil {
		return p, err
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.BlockNumber = d.BlockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func initReserves(p entity.Pool, poolTokens gyroeclp.PoolTokensResp) ([]string, error) {
	reserveByToken := make(map[string]*big.Int)
	for idx, token := range poolTokens.Tokens {
		addr := hexutil.Encode(token[:])
		reserveByToken[addr] = poolTokens.Balances[idx]
	}

	reserves := make([]string, len(p.Tokens))
	for idx, token := range p.Tokens {
		r, ok := reserveByToken[token.Address]
		if !ok {
			return nil, gyroeclp.ErrReserveNotFound
		}
		reserves[idx] = r.String()
	}

	return reserves, nil
}
