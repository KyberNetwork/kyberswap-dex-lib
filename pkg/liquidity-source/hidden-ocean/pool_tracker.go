package hiddenocean

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

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
	extra, reserves, blockNumber, err := t.fetchPoolState(ctx, &p)
	if err != nil {
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.SwapFee = float64(extra.Fee)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber

	return p, nil
}

// fetchPoolState retrieves current pool state via batched RPC calls.
func (t *PoolTracker) fetchPoolState(ctx context.Context, p *entity.Pool) (Extra, []string, uint64, error) {
	var (
		slot0     Slot0
		liquidity *uint256.Int
		fee       *uint256.Int
		rangeInfo RangeInfo
		balance0  *uint256.Int
		balance1  *uint256.Int
	)

	poolAddr := common.HexToAddress(p.Address)
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: methodSlot0,
	}, []any{&slot0}).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: methodLiquidity,
	}, []any{&liquidity}).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: methodFee,
	}, []any{&fee}).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: methodGetRange,
	}, []any{&rangeInfo}).AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: p.Tokens[0].Address,
		Method: erc20MethodBalanceOf,
		Params: []any{poolAddr},
	}, []any{&balance0}).AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: p.Tokens[1].Address,
		Method: erc20MethodBalanceOf,
		Params: []any{poolAddr},
	}, []any{&balance1}).TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.Address,
			"error": err,
		}).Error("failed to aggregate RPC calls")
		return Extra{}, nil, 0, err
	}

	extra := Extra{
		SqrtPriceX96: slot0.SqrtPriceX96,
		Liquidity:    liquidity,
		Fee:          uint32(fee.Uint64()),
		SqrtPaX96:    rangeInfo.SqrtPaX96,
		SqrtPbX96:    rangeInfo.SqrtPbX96,
	}

	reserves := []string{
		balance0.Dec(),
		balance1.Dec(),
	}

	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	logger.WithFields(logger.Fields{
		"pool":        p.Address,
		"blockNumber": blockNumber,
	}).Info("fetched pool state")

	return extra, reserves, blockNumber, nil
}
