package lido

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

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
	log := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})

	log.Infof("[Lido] Start getting new pool's state")

	extra, err := d.getPoolExtra(ctx, p)
	if err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to getPoolExtra")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to json marshal extra")
		return entity.Pool{}, err
	}

	reserves, err := d.getPoolReserves(ctx, p)
	if err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to getPoolReserves")
		return entity.Pool{}, err
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	log.Infof("[Lido] Finish getting new state of pool")

	return p, nil
}

func (d *PoolTracker) getPoolExtra(ctx context.Context, p entity.Pool) (Extra, error) {
	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var stEthPerToken, tokensPerStEth *big.Int

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    wstETHABI,
		Target: p.Address,
		Method: wstETHMethodStEthPerToken,
		Params: nil,
	}, []interface{}{&stEthPerToken})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    wstETHABI,
		Target: p.Address,
		Method: wstETHMethodTokensPerStEth,
		Params: nil,
	}, []interface{}{&tokensPerStEth})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Error("failed to process tryAggregate")
		return Extra{}, err
	}

	extra := Extra{
		StEthPerToken:  stEthPerToken,
		TokensPerStEth: tokensPerStEth,
	}

	return extra, nil
}

func (d *PoolTracker) getPoolReserves(ctx context.Context, p entity.Pool) (entity.PoolReserves, error) {
	var reserves = make([]*big.Int, len(p.Tokens))

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	for i, token := range p.Tokens {
		if token.Address == p.GetLpToken() {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: token.Address,
				Method: erc20MethodTotalSupply,
				Params: nil,
			}, []interface{}{&reserves[i]})
		} else {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: token.Address,
				Method: erc20MethodBalanceOf,
				Params: []interface{}{common.HexToAddress(p.Address)},
			}, []interface{}{&reserves[i]})
		}
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Error("failed to process tryAggregate")
		return entity.PoolReserves{}, err
	}

	poolReserves := make(entity.PoolReserves, len(reserves))
	for i := range reserves {
		poolReserves[i] = reserves[i].String()
	}

	return poolReserves, nil
}
