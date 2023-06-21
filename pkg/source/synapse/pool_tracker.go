package synapse

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[Synapse] Start getting new state of pool: %v", p.Address)

	var (
		lpSupply    *big.Int
		swapStorage SwapStorage
		balances    = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    swapFlashLoanABI,
			Target: p.Address,
			Method: poolMethodGetTokenBalance,
			Params: []interface{}{uint8(i)},
		}, []interface{}{&balances[i]})
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    swapFlashLoanABI,
		Target: p.Address,
		Method: poolMethodSwapStorage,
		Params: nil,
	}, []interface{}{&swapStorage})

	lpToken := p.GetLpToken()
	calls.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: lpToken,
		Method: erc20MethodTotalSupply,
		Params: nil,
	}, []interface{}{&lpSupply})

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to process RPC call")
		return entity.Pool{}, err
	}

	extra := Extra{
		InitialA:     swapStorage.InitialA.String(),
		FutureA:      swapStorage.FutureA.String(),
		InitialATime: swapStorage.InitialATime.Int64(),
		FutureATime:  swapStorage.FutureATime.Int64(),
		SwapFee:      swapStorage.SwapFee.String(),
		AdminFee:     swapStorage.AdminFee.String(),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	reserves := make(entity.PoolReserves, len(balances)+1)
	for i, balance := range balances {
		reserves[i] = balance.String()
	}
	reserves[len(balances)] = lpSupply.String()

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Synapse] Finish updating state of pool: %v", p.Address)

	return p, nil
}
