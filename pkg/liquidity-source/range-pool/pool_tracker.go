package rangepool

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState re-reads authoritative on-chain state each cycle (Strategy A):
// getRangePoolDynamicData (virtual + fact balances, rates, static fee, liveness) and
// the Vault's getPoolConfig (aggregate swap fee) in one atomic multicall. Immutable
// scaling factors are carried in StaticExtra (written at discovery) and copied into
// Extra so the Stage-3 simulator can hand extra.Extra straight to base.NewPoolSimulator.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	start := time.Now()
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	dyn, cfg, minTrade, blockNumber, err := t.fetchState(ctx, p.Address)
	if err != nil {
		return p, err
	}
	if p.BlockNumber > blockNumber {
		logger.WithFields(logger.Fields{
			"pool_id":           p.Address,
			"pool_block_number": p.BlockNumber,
			"data_block_number": blockNumber,
		}).Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	extra := Extra{
		Extra: &shared.Extra{
			StaticSwapFeePercentage:    uint256.MustFromBig(dyn.StaticSwapFeePercentage),
			AggregateSwapFeePercentage: uint256.MustFromBig(cfg.AggregateSwapFeePercentage),
			BalancesLiveScaled18:       shared.FromBigs(dyn.BalancesLiveScaled18),
			DecimalScalingFactors:      staticExtra.DecimalScalingFactors,
			TokenRates:                 shared.FromBigs(dyn.TokenRates),
		},
		VirtualBalances:      shared.FromBigs(dyn.VirtualBalances),
		MinimumTradeAmount:   uint256.MustFromBig(minTrade),
		IsPoolRegistered:     dyn.IsPoolRegistered,
		IsPoolInitialized:    dyn.IsPoolInitialized,
		IsPoolPaused:         dyn.IsPoolPaused,
		IsPoolInRecoveryMode: dyn.IsPoolInRecoveryMode,
		IsVaultPaused:        dyn.IsVaultPaused,
		IsHookStopped:        dyn.IsHookStopped,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	// Liveness gate: an unsafe pool is disabled by zeroing its reserves, which makes
	// base.PoolSimulator mark it paused (CalcAmountOut returns ErrPoolIsPaused).
	if extra.isLive() {
		p.Reserves = lo.Map(dyn.BalancesRaw, func(v *big.Int, _ int) string { return v.String() })
	} else {
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string { return "0" })
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.TotalSupply = dyn.TotalSupply.String()
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"pool_id":     p.Address,
		"block":       blockNumber,
		"duration_ms": time.Since(start).Milliseconds(),
	}).Info("Finished getting new pool state")

	return p, nil
}

func (t *PoolTracker) fetchState(
	ctx context.Context,
	poolAddress string,
) (*rangePoolDynamicDataABI, *poolConfigABI, *big.Int, uint64, error) {
	var (
		dyn      rangePoolDynamicDataResult
		cfg      poolConfigResult
		minTrade *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    rangePoolABI,
		Target: poolAddress,
		Method: poolMethodGetDynamicData,
	}, []any{&dyn})
	req.AddCall(&ethrpc.Call{
		ABI:    rangeVaultABI,
		Target: VaultAddress.Hex(),
		Method: vaultMethodGetPoolConfig,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&cfg})
	// getMinimumTradeAmount is a Vault-global immutable (same for every pool); reading
	// it in-band keeps the tracker the single source of the swap-path checks.
	req.AddCall(&ethrpc.Call{
		ABI:    rangeVaultABI,
		Target: VaultAddress.Hex(),
		Method: vaultMethodGetMinimumTradeAmount,
	}, []any{&minTrade})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, nil, 0, err
	}

	return &dyn.Data, &cfg.PoolConfig, minTrade, resp.BlockNumber.Uint64(), nil
}
