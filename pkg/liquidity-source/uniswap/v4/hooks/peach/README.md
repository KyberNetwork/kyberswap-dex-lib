# Peach Adaptive Pools

Shared handler for PeachAdaptivePoolDefaultHook (`Ramp`), PeachAdaptivePoolETHHook
(`RampETH`) and PeachAdaptivePoolRWAHook (`RwaRamp`). These immutable Uniswap v4
hooks enable only `beforeInitialize` and `beforeSwap` (permission bits `0x2080`).
Swaps use empty hook data, dynamic LP fees and no custom swap deltas. LP shares,
claims and deposits are handled by the contracts and do not change v4 swap math.

Source snapshot: [peachbuild/adaptive-pools@314ed215](https://github.com/peachbuild/adaptive-pools/tree/314ed2151367d3edb6f9045f05f5eb18567f5aa6).
The deployed Hook runtime was compared against the compiled snapshot after
normalizing constructor immutables; the immutable math helper was checked too.

## Deployments

| Chain | Contract | Address |
| --- | --- | --- |
| BSC (56) | Default | `0x4D1546682BC8252E52c1c1820F0AaA7694C9e080` |
| BSC (56) | ETH/native BNB | `0x1e4605837970EeF394662B1a9e88e09292636080` |
| BSC (56) | RWA | `0x98C819a522b91f76bAEAf499F3801e76890F2080` |
| Arc (5042) | Default | `0x8c960De62a26A688B72e6cbf644F46715642a080` |
| Arc (5042) | RWA | `0x3722d2840bFA4E9610d265DA08078d178bF46080` |

Use the existing v4 PoolManager discovery, tick tracker, native token wrapping,
protocol fee handling and swap simulation. No Peach Registry is required. Arc's
native USDC is not enabled by a Peach ETH Hook deployment in this handler.
Registration is scoped by both chain and Hook address; tracking also checks the
canonical PoolManager for that chain.

## Fees and refresh scheduling

`Track` batches both `currentFee` directions, both `autonomousFee` directions,
keeper state/expiry, pool floor/cap, absolute cap, global/per-pool pause deadlines,
Mid-block config and block timestamp in one multicall. The caller's block number
and state overrides apply to that entire call. A failed or inconsistent refresh
never replaces a valid snapshot.

The ordinary LP fee is the sampled autonomous policy (including any directional
premium), capped by the pool maximum. A nonzero, unexpired keeper fee is bounded
below by `max(cappedAutonomous / 2, floor)` and above by the pool cap. Expiry is
exclusive. Paused pools refuse quotes until both pause deadlines have passed.

The RWA autonomous policy depends on the MarketCalendar and session curve. We
sample its actual result rather than reproducing the calendar. `NextTrackAt`
requests per-second RWA refreshes and refreshes at keeper/pause expiry. The Kyber
backend must honor this optional schedule in addition to normal event tracking.
Ordinary fees remain subject to snapshot freshness: session/calendar/config
changes after the snapshot can change execution fees. Scheduling does not predict
future transactions or guarantee a quote's execution price.

## Mid-block simulation

Following `hooks/volatile`, a fresh tracked simulator assumes execution begins in
a new block. Its first accepted swap establishes the pre-swap sqrt-price reference
and pays no surcharge. Subsequent accepted swaps in the same simulated block use
that opening reference and the pool's current simulated pre-swap price. Speculative
quotes are pure; only `UpdateBalance(SwapInfo)` commits the reference. Clones own
all mutable state. A new tracking snapshot resets the simulated observation,
including after configuration changes.

The exact `MidBlockFeeMath` integer algorithm is ported using full-width squared
uint160 prices and a capped integer search. There is no tick approximation:

- `deviation = floor(abs(currentPrice / openingPrice - 1) * 1e6)`;
- only the return-toward-opening direction pays, when deviation meets the threshold;
- `extra = min(floor(deviation * captureFractionE6 / 1e6), midCap)`;
- `LP fee = min(ordinary fee + extra, poolCap)`.

The direction test uses the pre-swap price; crossing the opening price during the
swap does not reprice part of that swap. Disabling Mid-block avoids observation
state entirely. We do **not** predict external transaction ordering. An external
trade preceding our execution in the same block can change the actual reference
and surcharge, so slippage protection remains necessary.

Gas overhead estimates are callback allowances: 70,000 gas before
swap, with an additional 220,000 for a subsequent Mid-block leg (the Solidity
helper's staticcall is capped at 200,000 gas). Parent v4 execution/tick costs are
added separately. Actual quoter measurements and fixed-block comparisons are
recorded with the test fixtures. Total gas remains a heuristic: for the large
1 BTC Arc fixture, the parent v4 estimate is 554,009 versus 758,724 measured by
the canonical Quoter; small-quote estimates are conservative. These measurements
are not an execution gas guarantee.

## Verification

`testdata/mid_block_vectors.json` contains the 384 reference vectors from the
pinned source snapshot, including uint160 extremes, exact thresholds, rounding,
directions and caps. Fixtures use their own `go.mod` to avoid downstream module
zip bloat. Tests also cover speculative quote purity, accepted sequences, clone
isolation, pauses, keeper expiry, RWA refresh, chain scope and coherent RPC state
overrides. Source Foundry Mid-block suites validate actual sequential swap
observations, resets, failed settlement, pool isolation and module fallback.

```sh
go test ./pkg/liquidity-source/uniswap/v4/hooks/peach/...
go test -race ./pkg/liquidity-source/uniswap/v4/hooks/peach/...
CI=true go test ./pkg/liquidity-source/uniswap/v4/...
CI=true go test ./pkg/msgpack
go generate ./pkg/msgpack/...
```

Merging the adapter does not enable production routes by itself. Kyber must
configure BSC and Arc discovery, refresh scheduling and production activation,
then verify the resulting API routes.

The fixed-block fixtures contain 40 canonical Quoter cases at BSC block
129095585 and Arc block 27252638: 17 positive-output comparisons match exactly,
11 tiny inputs round to zero (refused by the simulator), and 12 quotes exhaust
liquidity. The Quoter requires a full fill; the parent v4 simulator can return a
partial fill with nonzero remaining input. BSC's native BNB fixture is at its
lower tick, so its zero-for-one direction has no meaningful executable liquidity
at that snapshot. Source Solidity verification ran all 384 math vectors plus
sequential observation/rollback tests (19 Mid-block tests, zero failures).

Online `Track` calls also decoded all five deployments successfully. All v4 and
msgpack tests pass with `CI=true`, the repository's switch for live-network tests.
Without that switch, the unrelated NAVJIT `TestLive_PluginVsV4Quoter` exceeds its
5 bps tolerance; the same failure reproduces on unmodified upstream main
`cce9e1035a288a3698fa4b8cb2f71490099eb8fc`. No NAVJIT code is changed here.
