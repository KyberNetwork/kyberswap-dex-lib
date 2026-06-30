package lazy

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	ringswap "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ringswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	pairMethodGetReserves = "getReserves"
	pairMethodBalanceOf   = "balanceOf"
)

type (
	PoolTracker struct {
		config       *ringswap.Config
		ethrpcClient *ethrpc.Client
		logDecoder   uniswapv2.ILogDecoder
	}
)

var _ = pooltrack.RegisterFactoryCE(ringswap.DexType, NewPoolTracker)

func NewPoolTracker(
	config *ringswap.Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logDecoder:   uniswapv2.NewLogDecoder(),
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	if err := validateTokens(p.Tokens); err != nil {
		return p, err
	}

	rpc := newRPCData()
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, p.Tokens, rpc)

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	if p.BlockNumber > resp.BlockNumber.Uint64() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":           p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": resp.BlockNumber.Uint64(),
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	logger.
		WithFields(
			logger.Fields{
				"pool_id":          p.Address,
				"old_reserve":      p.Reserves,
				"old_block_number": p.BlockNumber,
				"new_block_number": resp.BlockNumber,
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return buildPoolState(p, rpc, resp.BlockNumber)
}

func validateTokens(tokens []*entity.PoolToken) error {
	if len(tokens) < 4 {
		return errors.New("invalid number of tokens")
	}
	originalToken0, fwToken0 := tokens[0], tokens[2]
	originalToken1, fwToken1 := tokens[1], tokens[3]
	if (originalToken0.Address == fwToken0.Address) || (originalToken1.Address == fwToken1.Address) {
		return errors.New("waiting for fetching origin token address")
	}
	return nil
}

type rpcData struct {
	reservesResult   uniswapv2.ReserveData
	originalReserve0 *big.Int
	originalReserve1 *big.Int
}

func newRPCData() *rpcData {
	return &rpcData{
		originalReserve0: new(big.Int),
		originalReserve1: new(big.Int),
	}
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress string, tokens []*entity.PoolToken, d *rpcData) {
	addFn(&ethrpc.Call{
		ABI:    *ringswap.UniswapV2PairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
	}, []any{&d.reservesResult})
	addFn(&ethrpc.Call{
		ABI:    *ringswap.UniswapV2PairABI,
		Target: tokens[0].Address,
		Method: pairMethodBalanceOf,
		Params: []any{common.HexToAddress(tokens[2].Address)},
	}, []any{&d.originalReserve0})
	addFn(&ethrpc.Call{
		ABI:    *ringswap.UniswapV2PairABI,
		Target: tokens[1].Address,
		Method: pairMethodBalanceOf,
		Params: []any{common.HexToAddress(tokens[3].Address)},
	}, []any{&d.originalReserve1})
}

func buildPoolState(p entity.Pool, d *rpcData, blockNumber *big.Int) (entity.Pool, error) {
	fwReserves := uniswapv2.ReserveData{
		Reserve0: d.reservesResult.Reserve0,
		Reserve1: d.reservesResult.Reserve1,
	}
	if d.originalReserve0 == nil {
		d.originalReserve0 = bignumber.ZeroBI
	}
	if d.originalReserve1 == nil {
		d.originalReserve1 = bignumber.ZeroBI
	}
	originalReserves := uniswapv2.ReserveData{
		Reserve0: d.originalReserve0,
		Reserve1: d.originalReserve1,
	}
	return updatePool(p, fwReserves, originalReserves, blockNumber)
}

func updatePool(p entity.Pool, fwReserves, originalReserves uniswapv2.ReserveData, blockNumber *big.Int) (entity.Pool, error) {
	extra, err := json.Marshal(&originalReserves)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{
		fwReserves.Reserve0.String(),
		fwReserves.Reserve1.String(),
		"1",
		"1",
	}

	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	p.Extra = string(extra)

	return p, nil
}
