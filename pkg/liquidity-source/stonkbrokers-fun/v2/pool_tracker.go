package stonkbrokersfunv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
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

func NewPoolTracker(cfg *Config, client *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: client}, nil
}

// GetNewPoolState re-reads the mutable launch state (vQuote/vToken/flags)
// and the buy-side oracle snapshot (direct feed or TWAP, per StaticExtra).
//
// Robinhood Chain quirk (chain 4663, confirmed live against
// https://rpc.mainnet.chain.robinhood.com): eth_blockNumber and the
// block.number OPCODE are two independent, non-comparable counters.
// Multicall3's own aggregate()/tryBlockAndAggregate() reads block.number
// INSIDE the contract, so `resp.BlockNumber` from an ethrpc Aggregate/
// TryAggregate call is in the OPCODE domain -- pinning a later eth_call's
// block tag to that value always fails with "-32000 metadata is not found",
// because eth_call's block tag is resolved in the eth_blockNumber domain
// only. (Confirmed: eth_call pinned to a fresh eth_blockNumber succeeds;
// pinned to the numerically-current block.number value fails.) This bit a
// previous version of this tracker silently: the failed pin degraded to a
// logged warning and a nil oracle reading, which is SAFE (the simulator
// correctly refuses to quote without a reading, see CalcAmountOut's
// ErrBadOracleAnswer) but made every buy on every pad permanently
// unquotable in production, defeating the integration.
//
// Fix: fetch the eth_blockNumber domain value ONCE via a plain
// eth_blockNumber call (ethrpc.Client.GetBlockNumber -- NOT the multicall
// aggregate's returned block), then pin EVERY call in this refresh
// (getLaunch + the oracle read) to that same value. This is also the
// domain the chain-head resolution and event logs downstream use
// (pkg/repository/block/block.go's IEVMClient.BlockNumber -- standard
// eth_blockNumber), so entity.Pool.BlockNumber stays comparable with the
// rest of the stack (AGENTS.md: use the block from Aggregate/
// TryBlockAndAggregate/an existing GetBlockNumber/established event-log
// state; pin dependent reads to it -- GetBlockNumber is exactly the
// documented alternative when Aggregate's own block is unusable).
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := logger.WithFields(logger.Fields{"poolAddress": p.Address, "dexID": t.config.DexID})

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	launchID, ok := new(big.Int).SetString(staticExtra.LaunchID, 10)
	if !ok {
		return p, ErrInvalidPoolTokens
	}

	ethBlockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return p, err
	}
	blockNumber := new(big.Int).SetUint64(ethBlockNumber)

	var res getLaunchResult
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: staticExtra.Pad, Method: methodGetLaunch, Params: []any{launchID}}, []any{&res})
	if _, err := req.Aggregate(); err != nil {
		return p, err
	}
	lr := res.Launch

	extra := Extra{
		VQuote:       uint256.MustFromBig(lr.VQuote),
		VToken:       uint256.MustFromBig(lr.VToken),
		SellsEnabled: lr.SellsEnabled,
		Armed:        lr.Armed,
		Graduated:    lr.Graduated,
		Bonded:       lr.Bonded,
		Aborted:      lr.Aborted,
	}

	// A failed oracle read at this point (after pinning to the verified-
	// correct eth_blockNumber domain) is NOT the expected case -- it means a
	// genuine RPC hiccup or an actually-broken feed, not routine staleness
	// (staleness is a timestamp comparison inside feedUsd8/DirectFeedUsd8/
	// TwapQuoteUsd8 at quote time, not a call failure here). We still don't
	// fail the whole refresh: vQuote/vToken/armed/etc. above are valid and
	// worth persisting even if the oracle leg is temporarily down, and
	// persisting an explicit Ok:false reading already makes the pool
	// untradeable end-to-end (CalcAmountOut's currentQuoteUsd8 returns
	// ErrBadOracleAnswer for Ok:false / nil) -- so this IS "mark the pool
	// untradeable" (not silent), just logged louder than before since it is
	// now an unexpected condition rather than an expected one.
	switch {
	case staticExtra.QuoteUsdFeed != "":
		reading, err := t.fetchDirectFeed(ctx, staticExtra.QuoteUsdFeed, blockNumber)
		if err != nil {
			lg.Errorf("stonkbrokers-fun-v2: direct feed read failed at block %s (pool marked untradeable): %v", blockNumber, err)
		} else if reading != nil && !reading.Ok {
			lg.Errorf("stonkbrokers-fun-v2: direct feed call reverted at block %s (pool marked untradeable)", blockNumber)
		}
		extra.DirectFeed = reading
	case staticExtra.TwapPool != "":
		reading, err := t.fetchTwap(ctx, staticExtra.TwapPool, staticExtra.EthUsdFeed, staticExtra.TwapWindowSecs, blockNumber)
		if err != nil {
			lg.Errorf("stonkbrokers-fun-v2: twap read failed at block %s (pool marked untradeable): %v", blockNumber, err)
		} else if reading != nil && !reading.Ok {
			lg.Errorf("stonkbrokers-fun-v2: twap call reverted at block %s (pool marked untradeable)", blockNumber)
		}
		extra.Twap = reading
	default:
		lg.Errorf("stonkbrokers-fun-v2: neither QuoteUsdFeed nor TwapPool set in StaticExtra")
	}

	return t.persist(p, extra, staticExtra, blockNumber), nil
}

func (t *PoolTracker) fetchDirectFeed(ctx context.Context, feed string, blockNumber *big.Int) (*OracleReading, error) {
	var (
		roundData struct {
			RoundId         *big.Int
			Answer          *big.Int
			StartedAt       *big.Int
			UpdatedAt       *big.Int
			AnsweredInRound *big.Int
		}
		decimals uint8
	)
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req.AddCall(&ethrpc.Call{ABI: aggregatorV3ABI, Target: feed, Method: "latestRoundData"}, []any{&roundData})
	req.AddCall(&ethrpc.Call{ABI: aggregatorV3ABI, Target: feed, Method: "decimals"}, []any{&decimals})
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	if len(resp.Result) < 2 || !resp.Result[0] || !resp.Result[1] {
		return &OracleReading{Ok: false}, nil
	}
	if roundData.Answer == nil || roundData.Answer.Sign() <= 0 {
		return &OracleReading{Ok: false}, nil
	}
	return &OracleReading{
		Answer:    uint256.MustFromBig(roundData.Answer),
		Decimals:  decimals,
		UpdatedAt: roundData.UpdatedAt.Uint64(),
		Ok:        true,
	}, nil
}

func (t *PoolTracker) fetchTwap(ctx context.Context, twapPool, ethUsdFeed string, window uint32, blockNumber *big.Int) (*TwapReading, error) {
	// observe returns TWO outputs, so go-ethereum takes the copyTuple path and
	// needs ONE struct destination whose field names match the ABI's output
	// names. Passing two separate pointers looks reasonable but unpacks into
	// nothing: the multicall reports success, the destinations stay empty, and
	// the len check below then mislabels a perfectly good pool as untradeable.
	var observed struct {
		TickCumulatives                    []*big.Int
		SecondsPerLiquidityCumulativeX128s []*big.Int
	}
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req.AddCall(&ethrpc.Call{
		ABI: twapPoolABI, Target: twapPool, Method: "observe",
		Params: []any{[]uint32{window, 0}},
	}, []any{&observed})
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	tickCumulatives := observed.TickCumulatives
	if len(resp.Result) < 1 || !resp.Result[0] || len(tickCumulatives) != 2 {
		ethReading, _ := t.fetchDirectFeed(ctx, ethUsdFeed, blockNumber)
		if ethReading == nil {
			ethReading = &OracleReading{Ok: false}
		}
		return &TwapReading{Ok: false, EthUsd: *ethReading}, nil
	}

	ethReading, err := t.fetchDirectFeed(ctx, ethUsdFeed, blockNumber)
	if err != nil {
		return nil, err
	}
	if ethReading == nil {
		ethReading = &OracleReading{Ok: false}
	}

	return &TwapReading{
		TickCumulativeOld: tickCumulatives[0].Int64(),
		TickCumulativeNow: tickCumulatives[1].Int64(),
		EthUsd:            *ethReading,
		Ok:                true,
	}, nil
}

func (t *PoolTracker) persist(p entity.Pool, extra Extra, staticExtra StaticExtra, blockNumber *big.Int) entity.Pool {
	extraBytes, _ := json.Marshal(extra)
	p.Extra = string(extraBytes)
	if extra.VToken != nil && extra.VQuote != nil {
		p.Reserves = entity.PoolReserves{extra.VToken.Dec(), extra.VQuote.Dec()}
	}
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}

	if isTerminal(extra, staticExtra) {
		// This launch can never be swapped through again, so stop feeding it to
		// path-finding permanently. Timestamp is 1 rather than time.Now() -- and
		// not 0, which pool-service's IsPoolActive special-cases as
		// always-active -- so the pool looks maximally stale and is archived
		// immediately instead of waiting out the inactive-duration window. Same
		// convention as flap and pons-v2.
		p.Reserves = entity.PoolReserves{"0", "0"}
		p.Timestamp = 1
		return p
	}

	p.Timestamp = time.Now().Unix()
	return p
}

// isTerminal reports whether a launch is permanently unswappable. Every flag it
// reads is one-way in StonkSafeLaunchpadV2: armed/aborted/graduated/bonded are
// each assigned exactly once and never cleared, deadline is written once in
// arm(), and eoaOnly lives in _modesOf, written once in createLaunch on a
// non-upgradeable pad.
//
// Deliberately NOT terminal: an unarmed launch (the creator can still arm it)
// and a stale oracle (the feed recovers).
func isTerminal(extra Extra, staticExtra StaticExtra) bool {
	switch {
	case extra.Aborted, extra.Bonded, extra.Graduated:
		return true
	case staticExtra.EoaOnly:
		// Unroutable for an aggregator. Discovery skips these now, but pools
		// indexed before that change still need parking.
		return true
	case !staticExtra.OpenEnded && extra.Armed &&
		uint64(time.Now().Unix()) >= staticExtra.Deadline:
		// Timer close: the window is past and time only moves forward. Only
		// meaningful once armed, since deadline is set by arm().
		return true
	}
	return false
}
