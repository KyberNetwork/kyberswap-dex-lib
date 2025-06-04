package litepsm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeLitePSM, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, sourcePool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	pool entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	defer func(startTime time.Time) {
		logger.
			WithFields(logger.Fields{
				"exchange": pool.Exchange,
				"address":  pool.Address,
				"duration": time.Since(startTime).Milliseconds(),
			}).
			Info("finished GetNewPoolState")
	}(time.Now())

	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"exchange": pool.Exchange,
			"error":    err,
		}).Error("can not unmarshal static extra")
		return entity.Pool{}, err
	}

	litePSM, err := t.getLitePSM(ctx, staticExtra.Psm.String(), overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"exchange": pool.Exchange,
			"error":    err,
		}).Error("get psm error")
		return entity.Pool{}, err
	}

	extra := Extra{
		LitePSM: *litePSM,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"exchange": pool.Exchange,
			"error":    err,
		}).Error("can not marshal extra")
		return entity.Pool{}, err
	}

	reserves, err := t.getReserves(ctx, pool, staticExtra, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"exchange": pool.Exchange,
			"error":    err,
		}).Error("get reserves error")
		return entity.Pool{}, err
	}

	pool.Reserves = []string{reserves[0].String(), reserves[1].String()}
	pool.Extra = string(extraBytes)
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (t *PoolTracker) getLitePSM(
	ctx context.Context,
	address string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*LitePSM, error) {
	var tIn, tOut *big.Int
	var litePSM LitePSM

	req := t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    LitePSMABI,
			Target: address,
			Method: litePSMMethodTIn,
		}, []any{&tIn}).
		AddCall(&ethrpc.Call{
			ABI:    LitePSMABI,
			Target: address,
			Method: litePSMMethodTOut,
		}, []any{&tOut})

	if overrides != nil {
		req.SetOverrides(overrides)
	}
	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": DexTypeLitePSM,
			"error": err,
		}).Error("[getLitePSM] eth rpc call error")
		return nil, err
	}

	litePSM.TIn = number.SetFromBig(tIn)
	litePSM.TOut = number.SetFromBig(tOut)

	return &litePSM, nil
}

func (t *PoolTracker) getReserves(
	ctx context.Context,
	pool entity.Pool,
	staticExtra StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]*big.Int, error) {
	var daiReserve, gemReserve *big.Int
	req := t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: staticExtra.Dai.String(),
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{staticExtra.Psm},
		}, []any{&daiReserve}).
		AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: pool.Tokens[1].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{staticExtra.Pocket},
		}, []any{&gemReserve})

	if overrides != nil {
		req.SetOverrides(overrides)
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"exchange": pool.Exchange,
			"error":    err,
		}).Error("[getReserves] eth rpc call error")
		return nil, err
	}

	return []*big.Int{daiReserve, gemReserve}, nil
}
