package clear

import (
	"context"
	"math/big"
	"strings"
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

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

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
	logger.Infof("[Clear] Start getting new state of pool: %v", p.Address)

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Clear] failed to unmarshal static extra")
		return entity.Pool{}, err
	}

	if len(p.Tokens) < 2 {
		return entity.Pool{}, ErrPoolNotFound
	}

	// Get reserves by calling previewSwap with a small test amount
	// This tells us if the pool is active and has liquidity
	token0 := p.Tokens[0].Address
	token1 := p.Tokens[1].Address

	// Use a small test amount (1 unit of token0)
	testAmount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(p.Tokens[0].Decimals)), nil)

	var previewResult PreviewSwapResult

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    clearSwapABI,
		Target: staticExtra.SwapAddress,
		Method: methodPreviewSwap,
		Params: []any{
			common.HexToAddress(staticExtra.VaultAddress),
			common.HexToAddress(token0),
			common.HexToAddress(token1),
			testAmount,
		},
	}, []any{&previewResult.AmountOut, &previewResult.IOUs})

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Clear] failed to call previewSwap")

		// If the call fails, mark the pool as having zero reserves
		p.Reserves = entity.PoolReserves{zeroString, zeroString}
		p.Timestamp = time.Now().Unix()
		return p, nil
	}

	// If previewSwap returns 0, the pool might be paused or empty
	paused := previewResult.AmountOut == nil || previewResult.AmountOut.Sign() == 0

	// Estimate reserves based on preview result
	// Since Clear doesn't expose reserves directly, we use a placeholder
	// The actual swap amount is determined by previewSwap at execution time
	var reserve0, reserve1 string
	if paused {
		reserve0 = zeroString
		reserve1 = zeroString
	} else {
		// Use a large placeholder to indicate liquidity is available
		// The actual amounts come from previewSwap during pricing
		reserve0 = "1000000000000000000000000" // 1M tokens placeholder
		reserve1 = "1000000000000000000000000"
	}

	// Ensure AmountOut is not nil before converting
	amountOut := previewResult.AmountOut
	if amountOut == nil {
		amountOut = big.NewInt(0)
	}

	extra := Extra{
		Reserves: map[string]*uint256.Int{
			strings.ToLower(token0): uint256.MustFromBig(testAmount),
			strings.ToLower(token1): uint256.MustFromBig(amountOut),
		},
		Paused: paused,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Clear] failed to marshal extra")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserve0, reserve1}
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Clear] Finish getting new state of pool: %v", p.Address)

	return p, nil
}
