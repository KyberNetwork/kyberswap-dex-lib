package ironstable

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/timer"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	finish := timer.Start(fmt.Sprintf("[%s] get new pool state", d.cfg.DexID))
	defer finish()

	var (
		swapStorage        SwapStorage
		tokenBalances      []*big.Int
		lpTokenTotalSupply *big.Int
	)

	req := d.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    ironSwap,
			Target: p.Address,
			Method: ironSwapMethodSwapStorage,
			Params: nil,
		}, []interface{}{&swapStorage}).
		AddCall(&ethrpc.Call{
			ABI:    ironSwap,
			Target: p.Address,
			Method: ironSwapMethodGetTokenBalances,
			Params: nil,
		}, []interface{}{&tokenBalances}).
		AddCall(&ethrpc.Call{
			ABI:    erc20,
			Target: p.GetLpToken(),
			Method: erc20MethodTotalSupply,
			Params: nil,
		}, []interface{}{&lpTokenTotalSupply})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("error call contracts")
		return entity.Pool{}, err
	}

	extra := Extra{
		InitialA:           swapStorage.InitialA.String(),
		FutureA:            swapStorage.FutureA.String(),
		InitialATime:       swapStorage.InitialATime.Int64(),
		FutureATime:        swapStorage.FutureATime.Int64(),
		SwapFee:            swapStorage.Fee.String(),
		AdminFee:           swapStorage.AdminFee.String(),
		DefaultWithdrawFee: swapStorage.DefaultWithdrawFee.String(),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not marshal extra")
		return entity.Pool{}, err
	}

	reserves := make(entity.PoolReserves, 0, len(tokenBalances))
	for _, tokenBalance := range tokenBalances {
		reserves = append(reserves, tokenBalance.String())
	}
	reserves = append(reserves, lpTokenTotalSupply.String())

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
