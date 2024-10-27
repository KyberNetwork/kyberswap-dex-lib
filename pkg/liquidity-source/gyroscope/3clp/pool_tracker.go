package gyro3clp

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
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
	if err := sonic.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	// call RPC
	rpcRes, err := t.queryRPC(ctx, p.Address, staticExtra.PoolID, staticExtra.Vault, p.Tokens, overrides)
	if err != nil {
		return p, err
	}

	var (
		swapFeePercentage, _ = uint256.FromBig(rpcRes.SwapFeePercentage)
		poolTokens           = rpcRes.PoolTokens
		pausedState          = rpcRes.PausedState
		blockNumber          = rpcRes.BlockNumber
	)

	// update pool

	extra := Extra{
		PoolTokenInfos:    t.initPoolTokenInfos(rpcRes.PoolTokenInfos),
		SwapFeePercentage: swapFeePercentage,
		Paused:            !isNotPaused(pausedState),
	}
	extraBytes, err := sonic.Marshal(extra)
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

func (t *PoolTracker) initPoolTokenInfos(rpcResponse []PoolTokenInfoResp) []PoolTokenInfo {
	ret := make([]PoolTokenInfo, len(rpcResponse))
	for idx, info := range rpcResponse {
		ret[idx] = PoolTokenInfo{
			Cash:            uint256.MustFromBig(info.Cash),
			Managed:         uint256.MustFromBig(info.Managed),
			LastChangeBlock: info.LastChangeBlock.Uint64(),
			AssetManager:    info.AssetManager.Hex(),
		}
	}

	return ret
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
	overrides map[common.Address]gethclient.OverrideAccount,
) (*rpcRes, error) {
	var (
		poolTokens        PoolTokensResp
		poolTokenInfos    = make([]PoolTokenInfoResp, len(tokens))
		swapFeePercentage *big.Int
		pausedState       PausedStateResp

		poolIDHash = common.HexToHash(poolID)
	)

	req := t.ethrpcClient.R().
		SetContext(ctx).
		SetRequireSuccess(true)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []interface{}{poolIDHash},
	}, []interface{}{&poolTokens})

	for idx, token := range tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    shared.VaultABI,
			Target: vault,
			Method: shared.VaultMethodGetPoolTokenInfo,
			Params: []interface{}{poolIDHash, common.HexToAddress(token.Address)},
		}, []interface{}{&poolTokenInfos[idx]})
	}

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
		PoolTokens:        poolTokens,
		PoolTokenInfos:    poolTokenInfos,
		SwapFeePercentage: swapFeePercentage,
		PausedState:       pausedState,
		BlockNumber:       res.BlockNumber.Uint64(),
	}, nil
}

func isNotPaused(pausedState PausedStateResp) bool {
	return time.Now().Unix() > pausedState.BufferPeriodEndTime.Int64() || !pausedState.Paused
}
