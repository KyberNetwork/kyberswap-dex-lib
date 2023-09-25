package nerve

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
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	log := logger.WithFields(logger.Fields{
		"liquiditySource": DexTypeNerve,
		"poolAddress":     p.Address,
	})
	log.Infof("Start getting new state of pool")

	var swapStorage SwapStorage

	getSwapStorageRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	getSwapStorageRequest.AddCall(&ethrpc.Call{
		ABI:    swapABI,
		Target: p.Address,
		Method: methodGetSwapStorage,
		Params: nil,
	}, []interface{}{&swapStorage})

	if _, err := getSwapStorageRequest.Call(); err != nil {
		log.Errorf("failed to get swap storage, err: %v", err)
		return entity.Pool{}, err
	}

	extra := Extra{
		InitialA:           swapStorage.InitialA.String(),
		FutureA:            swapStorage.FutureA.String(),
		InitialATime:       swapStorage.InitialATime.Int64(),
		FutureATime:        swapStorage.FutureATime.Int64(),
		SwapFee:            swapStorage.SwapFee.String(),
		AdminFee:           swapStorage.AdminFee.String(),
		DefaultWithdrawFee: swapStorage.DefaultWithdrawFee.String(),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		log.Errorf("failed to marshal extra data, err: %v", err)
		return entity.Pool{}, err
	}

	lpToken := swapStorage.LpToken.Hex()

	if len(lpToken) == 0 {
		err := fmt.Errorf("couldnt get lp token")
		log.Errorf(err.Error())
		return entity.Pool{}, err
	}

	balances := make([]*big.Int, len(p.Tokens))
	for i := range balances {
		balances[i] = Zero
	}
	var totalSupply *big.Int

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i := range p.Tokens {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    swapABI,
			Target: p.Address,
			Method: methodGetTokenBalance,
			Params: []interface{}{uint8(i)},
		}, []interface{}{&balances[i]})
	}
	// add totalSupply to reserves to maintain the old logic, will check logic (other part) why we need to add totalSupply
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: lpToken,
		Method: methodGetTotalSupply,
		Params: nil,
	}, []interface{}{&totalSupply})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		log.Errorf("failed to get reserve, err: %v", err)
		return entity.Pool{}, err
	}

	reserves := entity.PoolReserves{}
	for _, balance := range balances {
		reserves = append(reserves, balance.String())
	}
	reserves = append(reserves, totalSupply.String())

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.TotalSupply = totalSupply.String()
	p.Timestamp = time.Now().Unix()

	log.Infof("Finish getting new state")

	return p, nil
}
