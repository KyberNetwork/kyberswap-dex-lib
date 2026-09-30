package oneswap

import (
	"context"
	"fmt"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE0(DexTypeOneSwap, NewPoolTracker)

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[Oneswap] Start getting new state of pool %v", p.Address)

	var (
		swapStorage SwapStorage
		balances    = make([]*uint256.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    oneSwapABI,
		Target: p.Address,
		Method: poolMethodSwapStorage,
		Params: nil,
	}, []any{&swapStorage})

	calls.AddCall(&ethrpc.Call{
		ABI:    oneSwapABI,
		Target: p.Address,
		Method: poolMethodGetBalances,
		Params: nil,
	}, []any{&balances})

	resp, err := calls.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to aggregate pool data")
		return entity.Pool{}, err
	}

	var extra = Extra{
		InitialA:           swapStorage.InitialA.String(),
		FutureA:            swapStorage.FutureA.String(),
		InitialATime:       u256ToInt64(swapStorage.InitialATime),
		FutureATime:        u256ToInt64(swapStorage.FutureATime),
		SwapFee:            swapStorage.SwapFee.String(),
		AdminFee:           swapStorage.AdminFee.String(),
		DefaultWithdrawFee: swapStorage.DefaultWithdrawFee.String(),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	lpToken := swapStorage.LpToken.Hex()

	if len(lpToken) == 0 {
		err := fmt.Errorf("couldnt get lp token")
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Error(err.Error())
		return entity.Pool{}, err
	}

	var totalSupply *uint256.Int
	calls = d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: lpToken,
		Method: methodGetTotalSupply,
		Params: nil,
	}, []any{&totalSupply})

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to aggregate pool total supply")
		return entity.Pool{}, err
	}
	p.BlockNumber = resp.BlockNumber.Uint64()

	var reserves = make([]string, 0, len(balances)+1)
	for _, balance := range balances {
		reserves = append(reserves, balance.String())
	}
	reserves = append(reserves, totalSupply.String())

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Oneswap] Finish getting new state of pool %v", p.Address)

	return p, nil
}

func u256ToInt64(v *uint256.Int) int64 {
	if v == nil {
		return 0
	}

	return v.ToBig().Int64()
}
