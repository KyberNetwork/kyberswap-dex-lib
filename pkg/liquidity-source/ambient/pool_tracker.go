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
		staticExtra StaticExtra

		baseReserve  *big.Int
		quoteReserve *big.Int
		sqrtPriceX64 *big.Int
		liquidity    *big.Int
	)

	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, fmt.Errorf("could not json.Unmarshal StaticExtra: %w", err)
	}

	baseToken := common.HexToAddress(staticExtra.Base)
	quoteToken := common.HexToAddress(staticExtra.Quote)
	queryAddress := common.HexToAddress(t.cfg.QueryContractAddress)
	swapAddress := common.HexToAddress(t.cfg.SwapDexContractAddress)
	poolIdx, _ := new(big.Int).SetString(staticExtra.PoolIdx, 10)

	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	if baseToken == nativeTokenPlaceholderAddress {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    multicallABI,
			Target: t.cfg.MulticallContractAddress,
			Method: "getEthBalance",
			Params: []interface{}{swapAddress},
		}, []interface{}{&baseReserve})
	} else {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: baseToken.Hex(),
			Method: "balanceOf",
			Params: []interface{}{swapAddress},
		}, []interface{}{&baseReserve})
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: quoteToken.Hex(),
		Method: "balanceOf",
		Params: []interface{}{swapAddress},
	}, []interface{}{&quoteReserve})

	// https://docs.ambient.finance/developers/query-contracts/crocquery-contract#pool-price
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    queryABI,
		Target: queryAddress.Hex(),
		Method: "queryPrice",
		Params: []interface{}{baseToken, quoteToken, poolIdx},
	}, []interface{}{&sqrtPriceX64})

	// https://docs.ambient.finance/developers/query-contracts/crocquery-contract#pool-liquidity
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    queryABI,
		Target: queryAddress.Hex(),
		Method: "queryLiquidity",
		Params: []interface{}{baseToken, quoteToken, poolIdx},
	}, []interface{}{&liquidity})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to call pool: %v, err: %v", p.Address, err)
		return p, err
	}

	encodedExtra, err := json.Marshal(Extra{
		Liquidity:    liquidity.String(),
		SqrtPriceX64: sqrtPriceX64.String(),
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return p, err
	}

	p.Extra = string(encodedExtra)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{(*baseReserve).String(), (*quoteReserve).String()}

	return p, nil
}
