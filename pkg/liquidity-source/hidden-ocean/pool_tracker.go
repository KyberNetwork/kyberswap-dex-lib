package hiddenocean

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

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	extra, reserves, blockNumber, err := fetchPoolState(ctx, t.ethrpcClient, p.Address)
	if err != nil {
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

// fetchPoolState retrieves current pool state via batched RPC calls.
// Shared between pool_list_updater and pool_tracker.
func fetchPoolState(
	ctx context.Context, ethrpcClient *ethrpc.Client, poolAddr string,
) (Extra, []string, uint64, error) {
	var (
		sqrtPriceX96 *big.Int
		tick         *big.Int
		unlocked     bool
		liquidity    *big.Int
		fee          *big.Int
		sqrtPaX96    *big.Int
		sqrtPbX96    *big.Int
		balance0     *big.Int
		balance1     *big.Int
	)

	// We need to know token addresses to query balanceOf. Read them first.
	var (
		token0Addr common.Address
		token1Addr common.Address
	)

	tokenReq := ethrpcClient.NewRequest().SetContext(ctx)
	tokenReq.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddr,
		Method: methodToken0,
	}, []any{&token0Addr})
	tokenReq.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddr,
		Method: methodToken1,
	}, []any{&token1Addr})

	if _, err := tokenReq.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"pool":  poolAddr,
			"error": err,
		}).Error("failed to fetch token addresses")
		return Extra{}, nil, 0, err
	}

	// Now batch-fetch all state in one call
	req := ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddr,
		Method: methodSlot0,
	}, []any{&sqrtPriceX96, &tick, &unlocked})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddr,
		Method: methodLiquidity,
	}, []any{&liquidity})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddr,
		Method: methodFee,
	}, []any{&fee})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddr,
		Method: methodGetRange,
	}, []any{&sqrtPaX96, &sqrtPbX96})

	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: token0Addr.Hex(),
		Method: erc20MethodBalanceOf,
		Params: []any{common.HexToAddress(poolAddr)},
	}, []any{&balance0})

	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: token1Addr.Hex(),
		Method: erc20MethodBalanceOf,
		Params: []any{common.HexToAddress(poolAddr)},
	}, []any{&balance1})

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool":  poolAddr,
			"error": err,
		}).Error("failed to aggregate RPC calls")
		return Extra{}, nil, 0, err
	}

	extra := Extra{
		SqrtPriceX96: uint256FromBigInt(sqrtPriceX96),
		Liquidity:    uint256FromBigInt(liquidity),
		Fee:          uint32(fee.Uint64()),
		SqrtPaX96:    uint256FromBigInt(sqrtPaX96),
		SqrtPbX96:    uint256FromBigInt(sqrtPbX96),
	}

	reserves := []string{
		balance0.String(),
		balance1.String(),
	}

	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	logger.WithFields(logger.Fields{
		"pool":        poolAddr,
		"blockNumber": blockNumber,
	}).Info("fetched pool state")

	return extra, reserves, blockNumber, nil
}
