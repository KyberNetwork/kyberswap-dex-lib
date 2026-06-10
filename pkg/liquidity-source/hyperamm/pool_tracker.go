package hyperamm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

// PoolTracker refreshes the mutable on-chain state of each HyperAMM pool.
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
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("hyperamm: refreshing pool %s", p.Address)

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	swapFeeModuleAddr := staticExtra.SwapFeeModule

	// ── Phase 1: read paused state, fair prices, base fee, miFactor ─────────
	var (
		isPaused       bool
		fairPrice0To1  *big.Int
		fairPrice1To0  *big.Int
		baseFeeBpsRaw  uint16
	)
	resp, err := t.ethrpcClient.NewRequest().
		SetContext(ctx).
		SetOverrides(overrides).
		AddCall(&ethrpc.Call{
			ABI:    hyperAMMABI,
			Target: p.Address,
			Method: "paused",
		}, []any{&isPaused}).
		AddCall(&ethrpc.Call{
			ABI:    hyperAMMSwapFeeModuleABI,
			Target: swapFeeModuleAddr,
			Method: "fairPriceInScale18",
			Params: []any{true},
		}, []any{&fairPrice0To1}).
		AddCall(&ethrpc.Call{
			ABI:    hyperAMMSwapFeeModuleABI,
			Target: swapFeeModuleAddr,
			Method: "fairPriceInScale18",
			Params: []any{false},
		}, []any{&fairPrice1To0}).
		AddCall(&ethrpc.Call{
			ABI:    hyperAMMSwapFeeModuleABI,
			Target: swapFeeModuleAddr,
			Method: "baseFeeBps",
		}, []any{&baseFeeBpsRaw}).
		Aggregate()
	if err != nil {
		return p, err
	}

	// ── Phase 2: read reserves and reference fees at the same block ──────────
	// Use 1e18 as reference amount for fee preview; the fee module accepts any
	// amount and we treat its output as a snapshot representative fee.
	refAmountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	zeroAddr := common.Address{}
	emptyBytes := []byte{}

	type swapFeeModuleData struct {
		FeeInBips *big.Int
		// internalContext bytes ignored
	}
	var (
		liquidity struct {
			Token0Amount *big.Int
			Token1Amount *big.Int
		}
		feeData0To1 struct{ Data swapFeeModuleData }
		feeData1To0 struct{ Data swapFeeModuleData }
	)
	if _, err := t.ethrpcClient.NewRequest().
		SetContext(ctx).
		SetOverrides(overrides).
		SetBlockNumber(resp.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    hyperAMMLensABI,
			Target: t.config.Lens,
			Method: "getAvailableLiquidity",
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&liquidity}).
		AddCall(&ethrpc.Call{
			ABI:    hyperAMMSwapFeeModuleABI,
			Target: swapFeeModuleAddr,
			Method: "previewSwapFeeInBips",
			Params: []any{
				common.HexToAddress(p.Tokens[0].Address),
				zeroAddr,
				refAmountIn,
				zeroAddr,
				emptyBytes,
			},
		}, []any{&feeData0To1}).
		AddCall(&ethrpc.Call{
			ABI:    hyperAMMSwapFeeModuleABI,
			Target: swapFeeModuleAddr,
			Method: "previewSwapFeeInBips",
			Params: []any{
				common.HexToAddress(p.Tokens[1].Address),
				zeroAddr,
				refAmountIn,
				zeroAddr,
				emptyBytes,
			},
		}, []any{&feeData1To0}).
		Aggregate(); err != nil {
		return p, err
	}

	fp01, _ := uint256.FromBig(fairPrice0To1)
	fp10, _ := uint256.FromBig(fairPrice1To0)

	var refFee01, refFee10 uint64
	if feeData0To1.Data.FeeInBips != nil {
		refFee01 = feeData0To1.Data.FeeInBips.Uint64()
	} else {
		refFee01 = uint64(baseFeeBpsRaw)
	}
	if feeData1To0.Data.FeeInBips != nil {
		refFee10 = feeData1To0.Data.FeeInBips.Uint64()
	} else {
		refFee10 = uint64(baseFeeBpsRaw)
	}

	// Fall back to baseFeeBps when fair price calls succeeded but fee preview
	// returned zero (shouldn't happen in production but guard against it).
	if refFee01 == 0 {
		refFee01 = uint64(baseFeeBpsRaw)
	}
	if refFee10 == 0 {
		refFee10 = uint64(baseFeeBpsRaw)
	}

	extraBytes, err := json.Marshal(Extra{
		FairPrice0To1: fp01,
		FairPrice1To0: fp10,
		BaseFeeBps:    baseFeeBpsRaw,
		RefFee0To1:    refFee01,
		RefFee1To0:    refFee10,
		IsPaused:      isPaused,
	})
	if err != nil {
		return p, err
	}

	token0Hex := hexutil.Encode(common.HexToAddress(p.Tokens[0].Address).Bytes())
	token1Hex := hexutil.Encode(common.HexToAddress(p.Tokens[1].Address).Bytes())
	_ = token0Hex
	_ = token1Hex

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Reserves = entity.PoolReserves{
		liquidity.Token0Amount.String(),
		liquidity.Token1Amount.String(),
	}

	return p, nil
}
