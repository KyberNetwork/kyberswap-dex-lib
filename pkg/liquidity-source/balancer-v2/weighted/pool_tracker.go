package weighted

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
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var ErrReserveNotFound = errors.New("reserve not found")

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

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

	if !staticExtra.BasePoolScanned {
		basePools, basePoolIds, err := t.scanBasePool(ctx, p.Address, p.Tokens)
		if err != nil {
			return p, err
		}

		if len(basePoolIds) > 0 {
			tokensByBasePool, err := t.scanUnderlyingTokens(ctx, p.Address, staticExtra.Vault, basePools, basePoolIds)
			if err != nil {
				return p, err
			}

			staticExtra.BasePools = tokensByBasePool
		}

		staticExtra.BasePoolScanned = true

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     DexType,
				"poolAddress": p.Address,
			}).Error(err.Error())

			return p, err
		}

		p.StaticExtra = string(staticExtraBytes)
	}

	// call RPC
	rpcRes, err := t.queryRPC(ctx, p.Address, staticExtra.PoolTypeVer, staticExtra.PoolID, staticExtra.Vault)
	if err != nil {
		return p, err
	}

	// update pool
	extra := Extra{
		SwapFeePercentage:         rpcRes.SwapFeePercentage,
		ProtocolSwapFeePercentage: rpcRes.ProtocolSwapFeePercentage,
		LastInvariant:             rpcRes.LastInvariant,
		TotalSupply:               rpcRes.TotalSupply,
		Paused:                    !isNotPaused(rpcRes.PausedState),
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

	reserves, err := t.initReserves(p, rpcRes.PoolTokens)
	if err != nil {
		return p, err
	}

	p.BlockNumber = rpcRes.BlockNumber
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
	poolTypeVer int,
	poolID string,
	vault string,
) (*rpcRes, error) {
	var (
		poolTokens                PoolTokens
		swapFeePercentage         *big.Int
		protocolSwapFeePercentage *big.Int
		pausedState               PausedState
		lastInvariant             *big.Int
		totalSupply               *big.Int
	)

	req := t.ethrpcClient.R().
		SetContext(ctx).
		SetRequireSuccess(true)

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []any{common.HexToHash(poolID)},
	}, []any{&poolTokens})

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

	if poolTypeVer == poolTypeVer1 {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetLastInvariant,
		}, []any{&lastInvariant})
	} else {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetInvariant,
		}, []any{&lastInvariant})
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodTotalSupply,
	}, []any{&totalSupply})

	req.AddCall(&ethrpc.Call{
		ABI:    shared.ProtocolFeesCollectorABI,
		Target: t.config.ProtocolFeesCollector,
		Method: protocolMethodGetSwapFeePercentage,
	}, []any{&protocolSwapFeePercentage})

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
		PoolTokens:                poolTokens,
		SwapFeePercentage:         uint256.MustFromBig(swapFeePercentage),
		ProtocolSwapFeePercentage: uint256.MustFromBig(protocolSwapFeePercentage),
		PausedState:               pausedState,
		LastInvariant:             uint256.MustFromBig(lastInvariant),
		TotalSupply:               uint256.MustFromBig(totalSupply),
		BlockNumber:               res.BlockNumber.Uint64(),
	}, nil
}

func (t *PoolTracker) scanBasePool(ctx context.Context, poolAddress string, tokens []*entity.PoolToken) ([]string, []string, error) {
	basePoolIds := make([]common.Hash, len(tokens))

	req := t.ethrpcClient.R().SetContext(ctx)
	for i, token := range tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: token.Address,
			Method: poolMethodGetPoolId,
			Params: []any{},
		}, []any{&basePoolIds[i]})
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": poolAddress,
		}).Error(err.Error())

		return nil, nil, err
	}

	basePools := make([]string, 0, len(tokens))
	validPoolIds := make([]string, 0, len(tokens))

	for i := range res.Result {
		basePoolId := basePoolIds[i].Hex()

		if res.Result[i] && basePoolId != shared.ZeroAddress {
			basePools = append(basePools, tokens[i].Address)
			validPoolIds = append(validPoolIds, basePoolId)
		}
	}

	return basePools, validPoolIds, nil
}

func (t *PoolTracker) scanUnderlyingTokens(
	ctx context.Context,
	poolAddress, vaultAddress string,
	basePools, basePoolIds []string,
) (map[string][]string, error) {
	var basePoolTokens = make([]PoolTokens, len(basePools))

	req := t.ethrpcClient.R().SetContext(ctx)
	for i, basePoolId := range basePoolIds {
		req.AddCall(&ethrpc.Call{
			ABI:    shared.VaultABI,
			Target: vaultAddress,
			Method: shared.VaultMethodGetPoolTokens,
			Params: []any{common.HexToHash(basePoolId)},
		}, []any{&basePoolTokens[i]})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": poolAddress,
		}).Error(err.Error())

		return nil, err
	}

	var result = make(map[string][]string, len(basePools))
	for i, poolTokens := range basePoolTokens {
		for _, token := range poolTokens.Tokens {
			tokenStr := strings.ToLower(token.Hex())
			result[basePools[i]] = append(result[basePools[i]], tokenStr)
		}
	}

	return result, nil
}

func isNotPaused(pausedState PausedState) bool {
	return time.Now().Unix() > pausedState.BufferPeriodEndTime.Int64() || !pausedState.Paused
}
