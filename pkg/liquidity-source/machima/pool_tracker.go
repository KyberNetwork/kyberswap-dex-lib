package machima

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

// PoolTracker runs as a ticks-based source: ticks are updated incrementally from the pool's own
// logs and state is read at that log's block, which is far cheaper than re-sweeping every tick.
//
// Machima is a UniV3 fork, so all of that is delegated to the UniV3 tracker rather than
// reimplemented. This tracker only layers on state UniV3 has no concept of: the ClankNow tax
// config, the pool deployment time, and the swap adapter's XMA sell floor. Those live in separate
// contracts and change without emitting any pool event, so the source must ALSO be configured with
// update_by_interval — that is the no-logs path handled in GetNewPoolState.
type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	v3           *uniswapv3.Tracker
}

var _ = pooltrack.RegisterFactoryCEG(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client) (*PoolTracker, error) {
	// AlwaysUseTickLens: the Machima subgraph indexes pools only and has no Tick entity, so the
	// bootstrap has to read ticks from the TickLens contract.
	v3, err := uniswapv3.NewTracker(&uniswapv3.Config{
		DexID:             cfg.DexID,
		TickLensAddress:   cfg.TickLensAddress,
		AlwaysUseTickLens: true,
	}, ethrpcClient, graphqlClient)
	if err != nil {
		return nil, err
	}

	return &PoolTracker{config: cfg, ethrpcClient: ethrpcClient, v3: v3}, nil
}

// BootstrapPoolState is the ticks-based first pass: every tick, plus full state.
func (t *PoolTracker) BootstrapPoolState(ctx context.Context, p entity.Pool,
	params poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	p, err := t.v3.BootstrapPoolState(ctx, p, params)
	if err != nil {
		return entity.Pool{}, err
	}
	return t.applyMachimaState(ctx, p, p.BlockNumber)
}

// FetchPoolTicks re-reads the known ticks from RPC.
//
// It cannot simply delegate: the UniV3 implementation rewrites Extra by marshalling a
// uniswapv3.Extra, which silently drops every Machima field. Losing them is not a decode error, it
// just leaves hasTax false — the pool would then quote with no tax at all. So the Machima half is
// carried across explicitly. Unlike the other two entry points there is no applyMachimaState after
// this, on purpose: bootstrap calls it right after BootstrapPoolState, which just read the tax.
func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	before, err := unmarshalExtra(p.Extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p, err = t.v3.FetchPoolTicks(ctx, p)
	if err != nil {
		return entity.Pool{}, err
	}

	after, err := unmarshalExtra(p.Extra)
	if err != nil {
		return entity.Pool{}, err
	}
	after.ProtocolState = before.ProtocolState

	if err = setExtra(&p, after); err != nil {
		return entity.Pool{}, err
	}
	return p, nil
}

// GetNewPoolState serves both triggers:
//   - with logs (event-driven): the UniV3 tracker applies the tick deltas carried by the logs and
//     reads slot0/liquidity/reserves at the log's block.
//   - without logs (interval): there are no tick changes to apply, so only pool state is refreshed
//     at latest. This is the path that keeps the tax config and XMA floor fresh.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	params poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	var (
		blockNumber uint64
		err         error
	)

	if len(params.Logs) > 0 {
		blockNumber = eth.GetBlockNumberFromLogs(params.Logs)
		if p, err = t.v3.GetNewPoolState(ctx, p, params); err != nil {
			return entity.Pool{}, err
		}
	} else if p, err = t.refreshStateKeepingTicks(ctx, p); err != nil {
		return entity.Pool{}, err
	}

	return t.applyMachimaState(ctx, p, blockNumber)
}

// refreshStateKeepingTicks re-reads slot0, liquidity and reserves at latest while leaving the tick
// list untouched — on the interval trigger there are no logs to derive tick changes from.
func (t *PoolTracker) refreshStateKeepingTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	rpcData, err := t.v3.FetchRPCData(ctx, &p, 0)
	if err != nil {
		return entity.Pool{}, errors.WithMessage(err, "fetch pool state")
	}
	if rpcData.Liquidity == nil || rpcData.Slot0.SqrtPriceX96 == nil || rpcData.Slot0.Tick == nil {
		return entity.Pool{}, errors.Errorf("machima pool %s returned incomplete state", p.Address)
	}

	extra, err := unmarshalExtra(p.Extra)
	if err != nil {
		return entity.Pool{}, err
	}

	extra.Tick = rpcData.Slot0.Tick
	extra.SqrtPriceX96 = rpcData.Slot0.SqrtPriceX96
	extra.Liquidity = rpcData.Liquidity
	if rpcData.TickSpacing != nil && rpcData.TickSpacing.Sign() > 0 {
		extra.TickSpacing = rpcData.TickSpacing.Uint64()
	}

	if err = setExtra(&p, extra); err != nil {
		return entity.Pool{}, err
	}
	if rpcData.Reserve0 != nil && rpcData.Reserve1 != nil {
		p.Reserves = entity.PoolReserves{rpcData.Reserve0.String(), rpcData.Reserve1.String()}
	}
	p.Timestamp = time.Now().Unix()

	return p, nil
}

// applyMachimaState layers the tax config, deployment time and XMA sell floor onto the Extra the
// UniV3 layer produced. Reads are pinned to blockNumber when there is one, so an event-driven
// refresh sees tax and pool state at the same block.
func (t *PoolTracker) applyMachimaState(ctx context.Context, p entity.Pool,
	blockNumber uint64) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, errors.WithMessage(err, "unmarshal staticExtra")
	}

	var (
		taxResult          struct{ Data TaxConfig }
		poolDeploymentTime *big.Int
		xmaSellPriceLimit  *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	}
	req.AddCall(&ethrpc.Call{
		ABI:    clankNowABI,
		Target: t.config.ClankNow,
		Method: methodGetTokenTax,
		Params: []any{common.HexToAddress(staticExtra.Token)},
	}, []any{&taxResult})
	req.AddCall(&ethrpc.Call{
		ABI: tokenABI, Target: staticExtra.Token, Method: methodPoolDeploymentTime,
	}, []any{&poolDeploymentTime})
	// The floor only ever applies to the pool whose traded token is XMA (see sqrtPriceLimit), so
	// there is no point reading it for every other pool.
	if t.config.SwapAdapter != "" && strings.EqualFold(staticExtra.Token, staticExtra.XMA) {
		req.AddCall(&ethrpc.Call{
			ABI: swapAdapterABI, Target: t.config.SwapAdapter, Method: methodXmaSellPriceLimit,
		}, []any{&xmaSellPriceLimit})
	}

	if _, err := req.Aggregate(); err != nil {
		// Same fallback the UniV3 layer applies to its own reads: a non-archive node cannot serve
		// a block that has been pruned, which happens while catching up after downtime. Reading at
		// latest is better than dropping the refresh entirely.
		if blockNumber > 0 && ticks.IsMissingTrieNodeError(err) {
			return t.applyMachimaState(ctx, p, 0)
		}
		logger.WithFields(logger.Fields{"dex": DexType, "pool": p.Address, "error": err}).
			Error("failed to fetch machima tax state")
		return entity.Pool{}, err
	}

	// Aggregate fails the whole batch if any sub-call reverts, but guard the decoded pointer
	// anyway: quoting a pool with a silently-zeroed tax is worse than not updating it at all.
	if poolDeploymentTime == nil {
		return entity.Pool{}, errors.Errorf("machima pool %s returned no poolDeploymentTime", p.Address)
	}

	extra, err := unmarshalExtra(p.Extra)
	if err != nil {
		return entity.Pool{}, err
	}

	extra.ProtocolState = ProtocolState{
		BuyTaxBps:          taxResult.Data.BuyTaxBps,
		SellTaxBps:         taxResult.Data.SellTaxBps,
		HasTax:             taxResult.Data.HasTax,
		PoolDeploymentTime: poolDeploymentTime.Uint64(),
	}
	if xmaSellPriceLimit != nil && xmaSellPriceLimit.Sign() > 0 {
		extra.XmaSellSqrtPriceLimit = xmaSellPriceLimit
	}

	if err = setExtra(&p, extra); err != nil {
		return entity.Pool{}, err
	}
	if blockNumber > 0 {
		p.BlockNumber = blockNumber
	}

	return p, nil
}

// unmarshalExtra decodes the Extra written by either layer. Machima's Extra is a superset of the
// UniV3 one and shares its JSON field names, so the UniV3 output round-trips through it.
func unmarshalExtra(raw string) (Extra, error) {
	var extra Extra
	if raw == "" {
		return extra, nil
	}
	if err := json.Unmarshal([]byte(raw), &extra); err != nil {
		return Extra{}, errors.WithMessage(err, "unmarshal extra")
	}
	return extra, nil
}

func setExtra(p *entity.Pool, extra Extra) error {
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return errors.WithMessage(err, "marshal extra")
	}
	p.Extra = string(extraBytes)
	return nil
}
