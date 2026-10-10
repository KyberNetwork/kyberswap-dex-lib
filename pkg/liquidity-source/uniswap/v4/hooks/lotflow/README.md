# LOTFLOW v1 on Robinhood Chain

Follow-up to [#1740](https://github.com/KyberNetwork/kyberswap-dex-lib/pull/1740).
Exchange: `uniswap-v4-lotflow` (package `hooks/lotflow`). The merged `navjit` package
(`uniswap-v4-navjit`, V15 venues) is unchanged.

**LOTFLOW v1 = the on-chain `NavJitHookV17` deployments below.** The contract names (`NavJitHookV17`,
`NavJitLegs`, `NavJitLegsMq`, `MqFactory` …), their errors and the recorded fixture case ids
(`navjit1-*`, `navjit2-*`) are on-chain or recorded facts and keep their names.

| LOTFLOW v1 deployment, chain 4663 | Address |
| --- | --- |
| [`NavJitHookV17` (stock baskets)](https://robinhoodchain.blockscout.com/address/0xb672a5d825F82dEA8574711d1dA696FcC48d2AE0#code) | `0xb672a5d825F82dEA8574711d1dA696FcC48d2AE0` |
| [`NavJitHookV17` #2 (WETH-constituent baskets)](https://robinhoodchain.blockscout.com/address/0x447184450e4615c2ec2545540B53F213fB9c2aE0#code) | `0x447184450e4615c2ec2545540B53F213fB9c2aE0` |

## What it is

A LOT is a redeemable basket. Its USDG/LOT Uniswap v4 hook mints constituents on a buy, or redeems
and sells them on a sell, and supplies just-in-time liquidity for the swap; standing ticks do not
describe the fill. V17 picks one of up to three admitted v3/v4 sources per constituent at the leg's
amount, for baskets of up to five constituents. Hook #2 also prices WETH-quoted constituents through
token/WETH and WETH/USDG v3 legs (TWAP via `observe`); the external pair is still USDG/LOT.

The adapter simulates the hook in integer arithmetic: basket rounding, the NAV guard, source
selection, every constituent leg, JIT placement, standing liquidity and the leftover unwind. Where a
probe's verdict rests on the gas envelope rather than exact maths, it takes the worse outcome
(`QuoteConservative`), so it may quote lower or refuse; it is not claimed to be exact in every case.

- `Track` resolves one block and reads all dependency state at it through the pool-service `ethrpc`
  client's multicall (`tryAggregate`; the client must have the chain's multicall contract set, as for
  every multicall source), installs the new state on the instance (the outer tracker calls
  `GetReserves` next) and publishes executable reserves. It runs on Robinhood Chain (4663) only
  (`ErrUnsupportedChain`, as #1740 refuses chains without a quoter); state overrides are rejected.
- Read failures are classified. A transport failure (HTTP, timeout, a node error that is not a
  revert) returns the error and publishes nothing. A read that reverts on chain (a NAV feed, the
  PoolManager LOT float, the WETH-basket ETH/USD or USDG/USD feed, any other load read), or a failed
  NAV check, publishes an empty market (reserves `0/0`, quotes refused), as #1740 publishes empty
  ladders, so a previously valid quote cannot survive it. Admission reads keep the contracts'
  semantics (reverted or short = failed read) and also return transport errors. A read cut short by
  the caller's context (including a deadline during the rate-limit back-off) is a transport error.
  A sub-call that fails inside `tryAggregate` is re-read on its own (one JSON-RPC batch of plain
  `eth_call`s at the pinned block), since a sub-call can also fail by running out of gas in a large
  batch: only the node's "execution reverted" counts as a revert, a success replaces the failure, any
  other error is a transport failure. Track logs its multicall counts, with a warning when reads
  missed the collect passes.
- `TrackedAt` is taken when `Track` starts, before its reads, so the `StaleCheck` age includes the
  time the reads took.
- The factory validates a decoded extra structurally (every pointer the simulator dereferences, big
  numbers in range, parameters within `setParams` bounds, WETH-basket sub-structures, oracle ring
  indices, and for every leg pool: slot0 tick consistent with its sqrtPrice, initialized ticks
  strictly ascending in-range multiples of the spacing, uint128 liquidity across the tick window) and
  treats an invalid one as untracked; every entry point nil-guards. `Track` validates the market it
  is about to publish and publishes the empty market instead of one that fails, so the tracking
  instance and a factory-built one agree. The venue price check mirrors the chain's
  `FullMath.mulDiv` revert on a zero denominator (`ErrMulDivZero`), and `quote`, the price limit,
  `GetReserves` and `Track`'s NAV check recover any other numeric panic into `ErrNumeric` (a refusal,
  or the empty market in `Track`). `FuzzHookExtra` (raw bytes) and `FuzzHookFields` (one leaf of a
  valid extra replaced) run their seeds and regression corpus in CI and count a recovered
  `ErrNumeric` as a failure.
- Quoting and executor metadata use the same NAV-band price limit (`HookPriceLimitProvider`): the
  hook's band less a 25 bps inward buffer, as introduced in #1740. A fill the limit would cut short
  is refused (`ErrPartialFill`), never quoted as a partial.
- **Why the NAV-band limit (fork evidence).** Replayed through Kyber's own router
  (`0x6131B5fae19EA4f9D964eAc0408E4408b66337b5`) and executor (`0x8F10B468b06c6FD214B65F87778827F7D113f996`)
  on anvil forks of chain 4663 (blocks 84,098,449 to 84,146,847; 30 rows, harness errors listed
  separately), comparing the earlier tick-derived limit with this one:
  - On the standing venue, the old limit (the standing position's tick) made every measured sell
    revert with `PriceLimitAlreadyExceeded` ($1, $100, $10k), because the hook repositions the pool
    to NAV less the sell spread, past that tick. Buys of $100 and $10k filled under both limits; at
    $250k and $300k the old limit reverted (decoded `PriceLimitAlreadyExceeded` at $300k; the $250k
    trace kept only inner frames). Under the NAV-band limit the $100 and $10k sells and the $250k and
    $300k buys filled.
  - Pool level: every fill under the NAV-band limit had the PoolManager pay out exactly the adapter's
    quote at $100-$10k (take == quote to the wei). That is measured at the tracked state: executed at
    the tracked timestamp. On the WETH-basket venue a later execution can take ~1 bps less (StaleCheck
    below). At $250k and $300k the adapter flagged the quote
    `Inexact` and the take exceeded it by 4.00 and 6.00 bps; the user received the quote and the
    excess went to Kyber's surplus. No row took less than the quote.
  - Route level, the user can receive 1 wei less than the take, because Kyber's executor keeps 1 wei
    of dust of the output token. Measured on a fork (block 84,237,756, lane A4 trace): on a $100 buy
    of each venue's LOT the PoolManager transferred exactly the quote to the executor
    (`0x8f10b468...f996`), the executor forwarded quote - 1 wei to the user, and its LOT balance went
    from 0 to 1. On the $100 sell the executor already held 1 wei of USDG and forwarded the full take.
    In all 30 fork gas runs every exact buy delivered quote - 1 wei and every sell exactly the quote.
    Read-only at block 84,240,315 the executor holds 0 of both LOTs and 1 wei of USDG, so the first
    buy of each LOT through it is 1 wei short. That buys after it are exact is inferred from the sell
    case, not measured. This is executor behaviour, not an adapter over-quote; the pool quote is
    unchanged.
  - On the WETH-basket venue (no standing position) both limits gave identical fills equal to the
    quote at $100 and $10k, both directions.
  - The refusals are conservative, not required for safety at the sizes measured: at a $350k buy
    the adapter refuses (`ErrPartialFill`), and the first PoolManager swap at its limit is the
    partial fill the model predicts (to 1 wei), but Kyber's executor then re-swaps the unspent input
    and the route completes. At $375k the adapter refuses (limit) while the chain fills at an
    extreme limit. The adapter gives up such fills; it does not over-quote them.
- A direction whose spread is at or past that buffered band (`spread >= bandBps - 25`) is refused by
  name (`ErrSpreadOutsideLimit`) and reports `0` reserve for that side. The live parameters
  (hook #1 buy/sell/band 25/100/200 bps, hook #2 25/200/600) are inside it. The guard is necessary,
  not sufficient: a buy edge is the executed cost plus the spread, so a spread just inside the bound
  can still be refused by the limit itself.
- `UpdateBalance` carries constituent pool state and the shared WETH/USDG conversion pool forward
  within one simulator instance; clones stay independent.
- Gas: reported per swap from the simulated swap's own features (`SwapGasFor`): an intercept per
  venue kind and direction, plus 45% of the probes' summed leg-gas upper bounds, plus 50k per
  initialized tick the executed legs cross, plus a full `PROBE_CALLER_GAS` (1,369,653) for every
  probe that may exhaust its stipend, rounded up to 0.1M and never below the old flat figure
  (3.9M two constituents, 6.9M five, 4.2M WETH-constituent basket, `SwapGas`). The hook's own figure
  subtracts the outer v4 base gas so it is not counted twice. What sets the transaction's gas limit is
  the probes, not gasUsed: every probe needs its 1.2M `PROBE_GAS` stipend available, so the minimum
  passing limit sits 0.7M-1.9M above gasUsed and grows with size.
  - Fork evidence (`testdata/gas/measured.json`, `TestSwapGasCoversForkMeasured`): 30 swaps on the
    two deployed venues through Kyber's router and executor on anvil forks of chain 4663 (blocks
    84,188,524-84,222,493), both directions, $1 up to the largest fill the adapter accepted at that block
    (largest measured: $231k sell / $399k buy on hook #1, ~$54k each way on hook #2; the accepted maximum
    moved between blocks, e.g. hook #1 sells $201k-$290k). Minimum passing limits 3.21M-6.76M;
    the flat figure was below it in 10 of 30 (e.g. a $231k sell crossing 36 ticks needs 6.76M, flat
    3.9M). The fitted part reports 1.12x-1.27x the minimum in every case. Fitted with a 12% margin;
    leaving any one of the eight fork sessions out of the fit, its cases stayed covered (worst 1.04x,
    the 36-tick sell; offline fit script, lane A4 logs).
  - Stipend-exhausting probes (`StipendProbes`): a probe the chain may run out of `PROBE_GAS` (the
    simulator skips it, or its leg-gas upper bound is within `InexactGasMargin` of the stipend, whether
    the maths fills or not) burns the whole stipend on chain (`NavJitLegs.sol:87`). The fit counts a
    filling probe at 45% of its capped bound (0.54M at most), so such a probe is now charged in full on
    top: 0.875M more, to `PROBE_CALLER_GAS` (`NavJitLegs.sol:99`, the stipend plus the frames around it).
    This term is not fitted; no fork gas case recorded a burned stipend. Offline evidence: in lane S's
    near-stipend sweep (fork @84,107,955) the 328 probes that ran out of gas measured 1.145M of frame gas
    each and all 328 are flagged by this rule (the fit counted 0.54M for each); in the P2-FIX fixtures,
    the 3 candidates whose re-measured leg needs >= 1.2M are counted, and the reported gas clears the
    fixture's own gasUsed on every fill (`TestGasFeaturesCountStipendExhaustingProbes`). The four
    Inexact cases of the fork table carry one such probe each, so they now report 1.26x-1.35x the
    minimum (standing-buy-max may carry one: up to 1.40x); the fitted part stays 1.12x-1.27x. These
    ratios describe the fork table only: in a separate set of 19 fresh fork cases (blocks
    84,248,041 and later), the largest standing-venue sell was Inexact and reports at least 1.55x its
    minimum passing limit (up to ~1.8x, depending on how many probes carry the charge), although its
    near-stipend probe did not run out of gas on chain. The charge errs high, so Inexact swaps can look
    more expensive to the router than they are; no measured case was under-reported.
  - Bounded: two venues, eight fork blocks, one Kyber executor version. Other basket shapes (more
    constituents, other candidate pools) are covered only by the flat floor and lane B's earlier
    5-constituent fork deployment (max 5.39M under 6.9M); size and state change the probes' gas, so the
    figure is an estimate, not a bound.

## RPC cost per `Track`

`Track` runs the loader's own read methods breadth-first in collect passes (a missing read is
recorded and answered with zeros; each pass is one multicall round), then the real load runs on the
warm cache through the same decode code. Oracle rings of up to 1,024 slots are fetched whole; a
larger ring by shadowing the TWAP binary search on a copy, 8 levels per round. Requests retry only
on rate limiting (3 attempts, 20 s each).

Read-only live, both venues at block 84,114,510 (`TestLoaderEquivalenceLive`), against the previous
one-`eth_call`-per-read loader at the same block:

| venue | before: HTTP requests / JSON-RPC calls / wall | after: HTTP requests / multicall sub-calls / wall | extras |
| --- | --- | --- | --- |
| LOT-V17 Review (hook #1, 2 constituents) | 133 / 994 / 38.8 s | 9 / 983 / 2.9 s (7 rounds) | byte-identical |
| bWALLETPIPEDOGNVDA (hook #2, WETH basket) | 233 / 1,768 / 66.0 s | 18 / 3,277 / 4.8 s (11 rounds) | byte-identical |

"Byte-identical" excludes `trackedAt`. The request counts exclude `eth_blockNumber` (one more when
`Track` resolves the block itself). Rounds: one for the hook's wiring, one per collect pass that found
new reads (6 / 7), and for the WETH basket 3 oracle-ring rounds. Its extra sub-calls are that ring
prefetch (1,589 `observations` reads), which replaces the binary search's sequential round trips. Offline,
`TestLoaderEquivalenceTape` replays those recorded reads (`testdata/rpc/`) and requires the same
byte-identical extra; `TestTrackRPCBudget` requires at most 25 requests per venue (11 / 20 offline,
`eth_blockNumber`, the pinned header and the re-read of the 1 / 2 sub-calls that revert included;
the live table above predates the re-read). Wall time depends on the RPC; these were measured on the public endpoint.

## Performance

No quote-path optimisation in v1. Each quote runs the hook's swap on `math/big` state (legs over
every constituent, candidate probing, NAV check); #1740 interpolates a 12-point ladder.
`go test -bench . -benchmem`, go1.26, darwin/arm64 (Apple M4 Max), one run:

| | 2 constituents | 5 constituents | WETH basket (3) |
| --- | --- | --- | --- |
| `BeforeSwap` buy $1k | 36 us, 867 allocs | 80 us, 1,792 allocs | 103 us, 1,392 allocs |
| `BeforeSwap` sell 10 LOT | 27 us, 718 allocs | 57 us, 1,419 allocs | 61 us, 1,090 allocs |
| simulator `CalcAmountOut` buy $1k | 37 us, 876 allocs | 82 us, 1,801 allocs | 99 us, 1,401 allocs |
| `GetReserves` (up to 8 quotes) | 68 us | 139 us | 162 us |

#1740 `navjit` `BenchmarkCalcAmountOut` on the same machine and run: 0.34-0.36 us, 15 allocs. A route
that evaluates LOTFLOW many times pays roughly 100-300 times #1740's per-call cost.

## Checks

```sh
CI=1 go test ./pkg/liquidity-source/uniswap/v4/hooks/lotflow/... -count=1 -race
CI=1 go test ./pkg/msgpack/... ./pkg/valueobject/... -count=1
go generate ./pkg/msgpack/... && git diff --exit-code pkg/msgpack
LOTFLOW_RPC=https://rpc.mainnet.chain.robinhood.com/ \
  go test ./pkg/liquidity-source/uniswap/v4/hooks/lotflow/ -run '^(TestAdapterLive|TestLoaderEquivalenceLive)$' -count=1 -v
go test ./pkg/liquidity-source/uniswap/v4/hooks/lotflow/ -run '^$' -fuzz '^FuzzHookFields$' -fuzztime 180s -fuzzminimizetime 2s
go test ./pkg/liquidity-source/uniswap/v4/hooks/lotflow/ -run '^$' -fuzz '^FuzzHookExtra$' -fuzztime 180s -fuzzminimizetime 2s
```

The fuzz targets need `-fuzzminimizetime`: with the 60 s default, Go's minimizer works on each new
~10 KB extra and the run shows long 0 execs/sec stretches.

The offline suites replay the recorded fork fixtures in `testdata/` (base, sequential/standing/refusal
and WETH-basket swaps) in raw units. The core keeps the recorded extreme-limit parity; the adapter
may refuse some of those sizes at its buffered NAV limit, and the fixture test counts those refusals
separately. Integration tests cover price-limit metadata, the spread guard, fresh reserves, invalid
NAV, stale state, pinned reads, rejected overrides, clone independence under `-race`, WETH
conversion-state updates and complete pool-simulator serialization through Kyber's msgpack registry.
The fixtures sit in a nested module (`testdata/go.mod`) so they are not part of the dex-lib module,
and are stored gzipped (`*.json.gz`, 3.6 MB instead of 20 MB); the loaders take the `.json` name and
read the `.gz` beside it. `TestFixtureCorpusDigest` pins the case count per suite (85 base, 43 a5,
14 MQ, 2 parity).

The live test (env-gated, read-only `eth_call` pinned to one block) quotes each size with the adapter
and with the on-chain V4Quoter. For above-float sells it gives the quoter a backed state override,
since that quoter does not prefund. It reports exact fills, lower quotes, joint reverts and
adapter-only refusals separately; PASS means zero over-quotes at that block, not that every sample
filled. `testdata/parity/summary.txt` is an earlier fork parity run; regenerating it needs Foundry
and the separate Lots contracts checkout (`KD_CONTRACTS`), which are not shipped here.

## Integration limits

- **Exact-input only.** `CalcAmountIn` (the v4 simulator calls hooks with `CalcOut=false`) is
  refused (`ErrExactOut`), so the router cannot use LOTFLOW for an exact-output leg. This is a
  capability #1740 had (it inverted its ladder). It costs no execution path that we have evidence
  for: Kyber's executor runs each v4 leg exact-input in its own unlock (measured on chain 4663,
  2026-10-07, K1 transactions from block 82,559,357), and the v4 simulator's own comment says the swap
  is always executed exact-in (`pool_simulator.go`, `CalcAmountIn`). The hook itself accepts
  exact-output (`NavJitHookV17._quote`), so supporting it later means simulating that branch.
- **USDG is currency0 on every venue these hooks can open; that is guaranteed by construction, not
  just observed.** `venueKey` sorts USDG below the LOT (`NavJitHookV17.sol` `venueKey`). `openVenue`
  accepts only `factory.isBasket(lot)`, and both factories set `isBasket` only on baskets deployed by
  `_deployAboveFloor`, at addresses at or above `ADDRESS_FLOOR = 2^159` (`BasketFactoryV15.sol`,
  `MqFactory.sol`; contracts source at d190fc1e). USDG `0x5fc5…d168` is below 2^159. A read-only
  check at block 84,136,095 returned `ADDRESS_FLOOR() = 2^159` from both hooks' factories
  (`0xe78b…DdFd`, `0x6033…3fB2`) and `usdg() = 0x5fc5…d168` from both hooks. The `ErrCurrencyOrder`
  refusal of the other order therefore never fires on these deployments. It stays as a defensive
  check.
- **Sells are priced prefunded** (settle before swap). That matches Kyber's executor, which we
  measured on 2026-10-07 on a Robinhood Chain fork settling every v4 leg in its own unlock before the
  swap. It is not enforced in code: a caller that swaps before settling can be refused on chain by the
  hook's PoolManager LOT float check (`SellExceedsFloat`) where the adapter quoted a fill. A
  #1740-style float refusal was not added because it would refuse nearly every sell on the
  WETH-constituent venue, whose PoolManager LOT float can be a few wei.
- Needs coordination with Kyber's routing service:
  - **Shared state.** The core `World` can model shared pools, but independent simulator instances do
    not share state through this adapter. A route that reuses an underlying candidate pool or the
    WETH/USDG conversion pool across another hop or basket needs dependency coordination or an
    overlap restriction in the service.
  - **Dependency invalidation.** A venue's quote depends on its constituents' pools, NAV feeds and
    TWAPs, not only on the LOT venue's own swap events. The dependency/invalidation map (`TrackerState.Invalidate` in
    `tracker.go`) is a model for that subscription, not an installed one.
  - **StaleCheck.** Route finding should run with `StaleCheck` on (60-second limit). WETH TWAPs use the
    tracked timestamp, so later execution can differ without a new trade. Measured on anvil forks of
    chain 4663 (DELTA4 review; same-pin A/B at block 84,269,062, plus runs at 84,254,925, 84,262,332 and
    84,266,049): on the WETH-basket venue (hook #2) a sell executed at the tracked timestamp took
    exactly the quote; executed 30 s later, with no trade in between, it took 978,547 raw USDG less on
    a $10k sell and 4,099,574 less on the $41k maximum (both ~1.0 bps), and 120 s later ~1.0 bps less
    again. The change is a step, not a drift, consistent with the 30-minute TWAP's integer tick moving
    by one (`TWAP_1800`, `MqFactory.sol:30`). The standing venue (hook #1) quoted 0.000 bps differently
    over 300 s of empty blocks (measured on buy quotes only). So "take == quote" holds at the tracked state; inside StaleCheck's 60 s
    hook #2 can deliver about 1 bps less than quoted, which the route's slippage bound (`minReturn`)
    has to absorb. Possible follow-up: evaluate the TWAP at quote time rather than at the tracked
    timestamp.
  - **Executor dust.** The executor keeps 1 wei of a swap's output token when it holds none
    (measured above), so a route's user receipt can be 1 wei below the pool-level quote. Kyber's
    builds carry a `minReturn` whose slippage allowance is far larger than 1 wei (the fork runs used
    99% of the quote); it should not be read as an adapter over-quote.
  - **Gas-limit propagation.** The per-swap gas above must reach the executed transaction's limit;
    probe stipends make the required limit higher than gas used.
- **Near-stipend probes (fork evidence).** A probe whose gas sits near the `PROBE_GAS` stipend is
  quoted at the worse of fill and skip (`QuoteConservative`). In a sweep on an anvil fork of chain 4663
  at block 84,107,955 (80 envelope cases on both deployed venues; cold, warm and mixed warm/cold access
  sets; 320 observations), it over-quoted 0 of 315 fills, and the chain's output was always inside the
  [all-skip, all-fill] bracket, including 12 mixed cases the policy does not evaluate. The argument
  rests on the deployed V17 rules (`MAX_CANDIDATES` 3, `MIN_TICK_SPACING` 10, `LEG_LIMIT_BPS` 500;
  immutable in these hooks, so there is no code guard) and on constituent legs using disjoint pools.
  A new deployment with other rules needs the sweep re-run.
- **An `Inexact` quote is a lower bound.** When `Inexact` is set the chain can return more than the
  quote, by up to the fill/skip spread (0.45-2.82% in that sweep), depending on which pools earlier
  route hops warmed. The simulator's lower gas bound for a probe assumes a cold, isolated transaction;
  inside a real swap the hook's own reads warm candidates first. That is harmless under the rules
  above.
- **For Kyber: a decoder panic shared by every v4 hook.** `uniswapv4.HookExtra.Unmarshal`
  (`hooks.go`) uses goccy/go-json v0.10.5, which `FuzzHookExtra` made panic (index out of range in
  `internal/decoder/context.go:49`) on a truncated extra with an escape inside a key. It does not
  panic in isolation; it panics when it follows a longer extra, because the decoder reads past the
  short input into its pooled buffer. `TestDecodeExtraRecoversDecoderPanic` reproduces it: 16-byte
  input `{"mArket":{"Mq\2` (corpus `8e6fd1838d839f3e`) after a ~10 KB extra panicked intermittently
  (order-dependent): between 36 and 101 of 200 attempts per run over 30 runs (15 by the DELTA4 review,
  43-99; 15 by lane A5, 36-101; offline, go1.26.2 darwin/arm64), and 0 of 600 under `-race`.
  LOTFLOW's factory recovers and treats the extra as untracked (`decodeExtra`); other hook factories
  do not. The inputs are in `testdata/fuzz/FuzzHookExtra/`.
- **Multicall contract: it must report the L2 block number.** Pool-service pins `Track` at the block
  number its own multicall returns (`uniswap/v4/pool_tracker.go:117`, `hookParam.BlockNumber =
  res.BlockNumber`). Robinhood Chain is Arbitrum-style: a contract's `block.number` is the L1 block
  number. Read-only at L2 block 84,175,294: `0x2cAC2D899eCC914d704FeaAE33ac1bF36277DaD1` (the address
  #1740's live tests and these tests use; it has `tryAggregate` and `getL1BlockNumber()` and no
  `aggregate3`, consistent with Offchain Labs' ArbMulticall2) returns `getBlockNumber()` = 84,175,294
  (= `ArbSys.arbBlockNumber()` = the header number), while canonical Multicall3
  `0xcA11bde05977b3631167028862bE2a173976CA11` returns 26,155,012 (= the header's `l1BlockNumber`).
  LOTFLOW itself reads no block number from a multicall (it uses `tryAggregate` and the header of the
  pinned block), but a pin taken from canonical Multicall3 is an L2 block about 68 days old. `Track`
  therefore refuses a caller's pin more than 60 s behind the chain head (`ErrStalePin`, nothing
  published; live at 26,155,012 it refuses before any state read). Configure `0x2cAC…DaD1`. Without
  any multicall contract the first round returns nothing and `Track` fails with a transport error.
- Source discovery/activation and Kyber-signed execution remain service steps. The executor evidence
  above used a local fork with a signer-role override; it does not establish production enablement.
