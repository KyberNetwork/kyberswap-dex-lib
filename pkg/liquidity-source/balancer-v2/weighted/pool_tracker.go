package weighted

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/vault"
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

	var (
		poolTokens        PoolTokens
		swapFeePercentage *big.Int
		pausedState       PausedState
	)

	// call RPC

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
		Method: poolMethodGetPausedState,
	}, []interface{}{&pausedState})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	// update pool

	extra := Extra{
		SwapFeePercentage: swapFeePercentage,
		Paused:            pausedState.Paused,
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

			return p, ErrReserveNotFound
		}

		reserves[idx] = r.String()
	}

	p.BlockNumber = res.BlockNumber.Uint64()
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}
