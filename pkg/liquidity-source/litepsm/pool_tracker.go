package litepsm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

	reserves, extra, err := t.fetchRPCData(ctx, &pool, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"exchange": pool.Exchange,
			"error":    err,
		}).Error("get psm error")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"exchange": pool.Exchange,
			"error":    err,
		}).Error("can not marshal extra")
		return entity.Pool{}, err
	}

	pool.Reserves = []string{reserves[0].String(), reserves[1].String()}
	pool.Extra = string(extraBytes)
	pool.Timestamp = time.Now().Unix()
	return pool, nil
}

func (t *PoolTracker) fetchRPCData(
	ctx context.Context,
	pool *entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]*big.Int, *Extra, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"exchange": pool.Exchange,
			"error":    err,
		}).Error("can not unmarshal static extra")
		return nil, nil, err
	}

	psm := common.HexToAddress(pool.Address)
	var tIn, tOut *big.Int
	reserves := make([]*big.Int, 2)
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    LitePSMABI,
		Target: pool.Address,
		Method: litePSMMethodTIn,
	}, []any{&tIn}).AddCall(&ethrpc.Call{
		ABI:    LitePSMABI,
		Target: pool.Address,
		Method: litePSMMethodTOut,
	}, []any{&tOut}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: pool.Tokens[1].Address,
		Method: abi.Erc20BalanceOfMethod,
		Params: []any{*lo.CoalesceOrEmpty(staticExtra.Pocket, staticExtra.GemJoin, &psm)},
	}, []any{&reserves[1]})
	if !staticExtra.IsMint {
		innerDai := pool.Tokens[0].Address
		if staticExtra.Dai != nil {
			innerDai = hexutil.Encode(staticExtra.Dai[:])
		}
		req = req.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: innerDai,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{*lo.CoalesceOrEmpty(staticExtra.GemJoin, &psm)},
		}, []any{&reserves[0]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": DexTypeLitePSM,
			"error": err,
		}).Error("[fetchRPCData] eth rpc call error")
		return nil, nil, err
	}

	if staticExtra.IsMint {
		reserves[0] = bignumber.TenPowInt(9 + pool.Tokens[0].Decimals)
	}

	var extra Extra
	if tIn.Sign() > 0 {
		extra.TIn = big256.FromBig(tIn)
	}
	if tOut.Sign() > 0 {
		extra.TOut = big256.FromBig(tOut)
	}
	return reserves, &extra, nil
}
