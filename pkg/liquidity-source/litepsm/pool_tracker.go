package litepsm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

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
				"dexID":             t.cfg.DexID,
				"poolsUpdatedCount": 1,
				"duration":          time.Since(startTime).Milliseconds(),
			}).
			Info("finished GetNewPoolState")
	}(time.Now())

	litePSM, err := t.getLitePSM(ctx, pool.Address, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": t.cfg.DexID,
			"error": err,
		}).Error("get psm error")
		return entity.Pool{}, err
	}

	extra := Extra{
		LitePSM: *litePSM,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": t.cfg.DexID,
			"error": err,
		}).Error("can not marshal extra")
		return entity.Pool{}, err
	}

	reserves, err := t.getReserves(ctx, pool, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": t.cfg.DexID,
			"error": err,
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
			ABI:    litePSMABI,
			Target: address,
			Method: litePSMMethodTIn,
			Params: nil,
		}, []interface{}{&tIn}).
		AddCall(&ethrpc.Call{
			ABI:    litePSMABI,
			Target: address,
			Method: litePSMMethodTOut,
			Params: nil,
		}, []interface{}{&tOut})

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
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]*big.Int, error) {
	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": t.cfg.DexID,
			"error": err,
		}).Error("can not unmarshal static extra")
		return nil, err
	}

	var daiReserve, gemReserve *big.Int

	req := t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: DAIAddress,
			Method: erc20MethodBalanaceOf,
			Params: []interface{}{common.HexToAddress(pool.Address)},
		}, []interface{}{&daiReserve}).
		AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: staticExtra.Gem.Address,
			Method: erc20MethodBalanaceOf,
			Params: []interface{}{staticExtra.Pocket},
		}, []interface{}{&gemReserve})
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	_, err = req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": DexTypeLitePSM,
			"error": err,
		}).Error("[getReserves] eth rpc call error")
		return nil, err
	}

	return []*big.Int{daiReserve, gemReserve}, nil
}
