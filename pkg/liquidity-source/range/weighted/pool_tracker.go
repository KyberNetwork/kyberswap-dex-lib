package weighted

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/rs/zerolog/log"
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

// Registered as the backup (per-pool, non-batched) tracker: the primary/batchable
// tracker is range/weighted/lazy.PoolTracker, which plans the same RPC calls (via
// AddRPCCalls/BuildPoolState below) into a pool.LazyRequest so pool-service's worker
// can merge them with other pools' calls into one multicall. This mirrors the split
// balancer/v3 weighted/stable/eclp already use.
var _ = pooltrack.RegisterBackupFactoryCE(DexType, NewPoolTracker)

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
	l := log.Ctx(ctx).With().Str("dex", DexType).Str("pool", p.Address).Logger()
	l.Info().Msg("Started getting new pool state")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	dyn, cfg, minTrade, blockNumber, err := t.fetchState(ctx, p.Address)
	if err != nil {
		return p, err
	}
	if p.BlockNumber > blockNumber {
		l.Info().
			Uint64("poolBlockNumber", p.BlockNumber).
			Uint64("dataBlockNumber", blockNumber).
			Msg("skip update: data block number is less than current pool block number")
		return p, nil
	}

	newP, err := BuildPoolState(p, &staticExtra, dyn, cfg, minTrade, blockNumber)
	if err != nil {
		return p, err
	}

	l.Info().
		Uint64("block", blockNumber).
		Int64("durationMs", time.Since(start).Milliseconds()).
		Msg("Finished getting new pool state")

	return newP, nil
}

func (t *PoolTracker) fetchState(
	ctx context.Context,
	poolAddress string,
) (*RangePoolDynamicDataABI, *PoolConfigABI, *big.Int, uint64, error) {
	var (
		dyn      RangePoolDynamicDataResult
		cfg      PoolConfigResult
		minTrade *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	AddRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, VaultAddress(t.config.ChainID), poolAddress, &dyn, &cfg, &minTrade)

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, nil, 0, err
	}

	// TryBlockAndAggregate tolerates per-call reverts, so err == nil does not mean every
	// read succeeded. All three reads are required to build valid state — bail if the
	// block number is missing or any call failed, rather than decoding zero/nil values.
	if resp.BlockNumber == nil || len(resp.Result) < 3 {
		return nil, nil, nil, 0, ErrIncompleteState
	}
	for _, ok := range resp.Result {
		if !ok {
			return nil, nil, nil, 0, ErrIncompleteState
		}
	}

	return &dyn.Data, &cfg.PoolConfig, minTrade, resp.BlockNumber.Uint64(), nil
}

// AddRPCCalls plans the three calls that make up a Range Pool's dynamic state:
// the pool's own getRangePoolDynamicData, and the Vault's getPoolConfig (aggregate
// swap fee is not part of the pool getter) and getMinimumTradeAmount (a Vault-global
// immutable, but read in-band so the tracker is the single source of the swap-path
// checks). addFn is generic over eager (*ethrpc.Request) and lazy (pool.LazyRequest)
// callers so range/weighted/lazy can reuse this unchanged.
func AddRPCCalls(
	addFn func(*ethrpc.Call, []any),
	vaultAddress common.Address,
	poolAddress string,
	dyn *RangePoolDynamicDataResult,
	cfg *PoolConfigResult,
	minTrade **big.Int,
) {
	addFn(&ethrpc.Call{
		ABI:    rangePoolABI,
		Target: poolAddress,
		Method: poolMethodGetDynamicData,
	}, []any{dyn})
	addFn(&ethrpc.Call{
		ABI:    rangeVaultABI,
		Target: hexutil.Encode(vaultAddress[:]),
		Method: vaultMethodGetPoolConfig,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{cfg})
	addFn(&ethrpc.Call{
		ABI:    rangeVaultABI,
		Target: hexutil.Encode(vaultAddress[:]),
		Method: vaultMethodGetMinimumTradeAmount,
	}, []any{minTrade})
}

// BuildPoolState assembles the updated entity.Pool from a completed state read. Shared
// by the backup (eager) and lazy (batched) trackers so their output can never diverge.
func BuildPoolState(
	p entity.Pool,
	staticExtra *StaticExtra,
	dyn *RangePoolDynamicDataABI,
	cfg *PoolConfigABI,
	minTrade *big.Int,
	blockNumber uint64,
) (entity.Pool, error) {
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

	return p, nil
}
