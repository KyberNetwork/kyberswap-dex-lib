package genericarm

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
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
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)
	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	var liquidityAsset common.Address
	var tradeRate0, tradeRate1, priceScale, withdrawsQueued, withdrawsClaimed, reserve0, reserve1 *big.Int
	calls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: t.config.ArmAddress,
		Method: "traderate0",
	}, []interface{}{&tradeRate0}).
		AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: t.config.ArmAddress,
			Method: "traderate1",
		}, []interface{}{&tradeRate1}).
		AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: t.config.ArmAddress,
			Method: "PRICE_SCALE",
		}, []interface{}{&priceScale}).
		AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: t.config.ArmAddress,
			Method: "withdrawsQueued",
		}, []interface{}{&withdrawsQueued}).
		AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: t.config.ArmAddress,
			Method: "withdrawsClaimed",
		}, []interface{}{&withdrawsClaimed}).
		AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: t.config.ArmAddress,
			Method: "liquidityAsset",
		}, []interface{}{&liquidityAsset}).
		AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: p.Tokens[0].Address,
			Method: "balanceOf",
			Params: []interface{}{common.HexToAddress(t.config.ArmAddress)},
		}, []interface{}{&reserve0}).
		AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: p.Tokens[1].Address,
			Method: "balanceOf",
			Params: []interface{}{common.HexToAddress(t.config.ArmAddress)},
		}, []interface{}{&reserve1})
	_, err := calls.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return p, err
	}

	extra := Extra{
		TradeRate0:       uint256.MustFromBig(tradeRate0),
		TradeRate1:       uint256.MustFromBig(tradeRate1),
		PriceScale:       uint256.MustFromBig(priceScale),
		WithdrawsQueued:  uint256.MustFromBig(withdrawsQueued),
		WithdrawsClaimed: uint256.MustFromBig(withdrawsClaimed),
		LiquidityAsset:   liquidityAsset,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		reserve0.String(),
		reserve1.String(),
	}

	p.Timestamp = time.Now().Unix()
	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)
	return p, nil
}
