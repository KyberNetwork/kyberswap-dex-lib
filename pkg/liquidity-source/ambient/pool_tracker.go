package ambient

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type PoolTracker struct {
	cfg          Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":     p.Address,
					"duration_ms": time.Since(startTime).Milliseconds(),
				},
			).
			Info("Finished getting new pool state")
	}()

	var (
		poolAddr           = common.HexToAddress(p.Address)
		queryAddress       = common.HexToAddress(t.cfg.QueryContractAddress)
		nativeTokenAddress = common.HexToAddress(t.cfg.NativeTokenAddress)

		extra Extra
	)

	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": t.cfg.DexID, "address": p.Address, "err": err}).
			Error("could not json.Unmarshal Extra")
		return p, fmt.Errorf("could not json.Unmarshal Extra: %w", err)
	}

	var (
		reserves = make([]*big.Int, len(p.Tokens)) // reserves[i] is corresponding to p.Tokens[i]

		tokenPairs    = make([]TokenPair, len(extra.TokenPairs))
		sqrtPriceX64s = make([]*big.Int, len(extra.TokenPairs)) // sqrtPriceX64s[i] is corresponding to tokenPairs[i]
		liquidities   = make([]*big.Int, len(extra.TokenPairs)) // liquidities[i] is corresponding to tokenPairs[i]
	)

	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	for i, token := range p.Tokens {
		tokenAddr := common.HexToAddress(token.Address)
		if tokenAddr == nativeTokenAddress {
			// native token reserve is the balance of the pool contract itself
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    multicallABI,
				Target: t.cfg.MulticallContractAddress,
				Method: "getEthBalance",
				Params: []interface{}{poolAddr},
			}, []interface{}{&reserves[i]})
		} else {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: tokenAddr.Hex(),
				Method: "balanceOf",
				Params: []interface{}{poolAddr},
			}, []interface{}{&reserves[i]})
		}
	}

	i := 0
	for pair, pairInfo := range extra.TokenPairs {
		tokenPairs[i] = pair

		// https://docs.ambient.finance/developers/query-contracts/crocquery-contract#pool-price
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    queryABI,
			Target: queryAddress.Hex(),
			Method: "queryPrice",
			Params: []interface{}{pair.Base, pair.Quote, pairInfo.PoolIdx},
		}, []interface{}{&sqrtPriceX64s[i]})

		// https://docs.ambient.finance/developers/query-contracts/crocquery-contract#pool-liquidity
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    queryABI,
			Target: queryAddress.Hex(),
			Method: "queryLiquidity",
			Params: []interface{}{pair.Base, pair.Quote, pairInfo.PoolIdx},
		}, []interface{}{&liquidities[i]})

		i++
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.
			WithFields(logger.Fields{"poolAddress": p.Address, "error": err}).
			Error("failed to call multical contract TryAggregate")
		return p, err
	}

	for i, pair := range tokenPairs {
		if liquidities[i] != nil {
			extra.TokenPairs[pair].Liquidity = liquidities[i].String()
		} else {
			logger.
				WithFields(logger.Fields{"poolAddress": p.Address}).
				Warnf("could not fetch liquidity for pair %s", pair)
		}
		if sqrtPriceX64s[i] != nil {
			extra.TokenPairs[pair].SqrtPriceX64 = sqrtPriceX64s[i].String()
		} else {
			logger.
				WithFields(logger.Fields{"poolAddress": p.Address}).
				Warnf("could not fetch sqrtPriceX64 for pair %s", pair)
		}
	}

	encodedExtra, err := json.Marshal(extra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"poolAddress": p.Address, "error": err}).
			Error("failed to marshal extra data")
		return p, err
	}

	p.Extra = string(encodedExtra)
	p.Timestamp = time.Now().Unix()
	for i := len(p.Reserves); i < len(p.Tokens); i++ {
		p.Reserves = append(p.Reserves, "")
	}
	for i, reserve := range reserves {
		p.Reserves[i] = reserve.String()
	}

	return p, nil
}
