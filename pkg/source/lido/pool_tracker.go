package lido

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE0(DexTypeLido, NewPoolTracker)

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

	extra, blockNumber, err := d.getPoolExtra(ctx, p)
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

	reserves, reserveBlockNumber, err := d.getPoolReserves(ctx, p, blockNumber)
	if err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to getPoolReserves")
		return entity.Pool{}, err
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	if reserveBlockNumber != nil {
		p.BlockNumber = reserveBlockNumber.Uint64()
	} else if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}

	log.Infof("[Lido] Finish getting new state of pool")

	return p, nil
}

func (d *PoolTracker) getPoolExtra(ctx context.Context, p entity.Pool) (Extra, *big.Int, error) {
	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var stEthPerToken, tokensPerStEth *big.Int

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    wstETHABI,
		Target: p.Address,
		Method: wstETHMethodStEthPerToken,
		Params: nil,
	}, []any{&stEthPerToken})
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    wstETHABI,
		Target: p.Address,
		Method: wstETHMethodTokensPerStEth,
		Params: nil,
	}, []any{&tokensPerStEth})

	response, err := rpcRequest.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Error("failed to process tryAggregate")
		return Extra{}, nil, err
	}

	extra := Extra{
		StEthPerToken:  stEthPerToken,
		TokensPerStEth: tokensPerStEth,
	}

	return extra, response.BlockNumber, nil
}

func (d *PoolTracker) getPoolReserves(
	ctx context.Context, p entity.Pool, blockNumber *big.Int,
) (entity.PoolReserves, *big.Int, error) {
	var reserves = make([]*big.Int, len(p.Tokens))

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	if blockNumber != nil {
		rpcRequest.SetBlockNumber(blockNumber)
	}

	for i, token := range p.Tokens {
		if token.Address == p.GetLpToken() {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: token.Address,
				Method: erc20MethodTotalSupply,
				Params: nil,
			}, []any{&reserves[i]})
		} else {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: token.Address,
				Method: erc20MethodBalanceOf,
				Params: []any{common.HexToAddress(p.Address)},
			}, []any{&reserves[i]})
		}
	}

	response, err := rpcRequest.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Error("failed to process tryAggregate")
		return entity.PoolReserves{}, nil, err
	}

	poolReserves := make(entity.PoolReserves, len(reserves))
	for i := range reserves {
		poolReserves[i] = reserves[i].String()
	}

	return poolReserves, response.BlockNumber, nil
}
