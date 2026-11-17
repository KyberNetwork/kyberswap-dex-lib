# ThogAMM schema-8 / model-3 adapter

ThogAMM is a portfolio-inventory AMM on Monad (chain ID 143). Integrate the proxy
`0x80c74517bcc2d67ffe02d3ed886796272f647210`. The implementation observed at block
122916130 is `0xb108a1daf3c9ba52778b97a15ebdfa3f6f59b506`. Pool ID:
`0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc`.

[Integration guide and integer specification](https://github.com/thogard785/thog-amm-integration-guide/blob/main/THOGAMM_AGGREGATOR_INTEGRATION.md)
· [Proxy on Monadscan](https://monadscan.com/address/0x80c74517bcc2d67ffe02d3ed886796272f647210)
· [Implementation](https://monadscan.com/address/0xb108a1daf3c9ba52778b97a15ebdfa3f6f59b506)
· [Reference Rust model](https://github.com/thogard785/thog-amm-poll-sdk/tree/v0.6.1/model)

## Live quote age

The registered factory honors `FactoryParams.Opts.StaleCheck`. Live routing must
set it to `true`. Every quote evaluates the unchanged snapshot at:

```text
elapsed blocks = ceil((nowMs - capturedAtMs) / blockTimeMs)
quote block = snapshot block + elapsed blocks + 1
quote age = quote block - posted price block
age spread = quote age × on-chain widening per block
```

The default block interval is **300 ms**, and the last block is the existing
inclusion lookahead. The captured contract policy supplies the widening and
maximum age; the adapter never assumes that an operator has published a new
policy. At 0.50 bp/block and a published maximum age of 15, the last executable
modeled age has 7.50 bp of age spread. Prices already old at capture have less
remaining lifetime. This is a model using the nominal block interval, not a
prediction of subsequent prices or an exact measurement of the current head.

`ErrStaleSnapshot` identifies an estimated execution block beyond the on-chain
maximum age, a missing capture time or one more than a second in the future,
or the wall-clock backstop. The
default backstop is **15,000 ms**, independent of the on-chain maximum age.
Capture time is the **start** of the successful RPC attempt, so request latency
counts toward age. Old stored rows without millisecond capture time use
`entity.Pool.Timestamp` in seconds; creating a simulator never resets their age.
When the lens provides a block timestamp, the effective capture time cannot be
later than the end of that second, so a lagging provider cannot freshen an old
head. Second-resolution timestamps and nominal block time limit the estimate's
precision; final execution simulation remains authoritative.

For indexing (`StaleCheck=false`), elapsed-time and maximum-age rejection are
skipped. The indexer uses snapshot block + 1 and retains the curve, pause, side,
and inventory checks. Unlike the previous adapter, an expired on-chain price can
therefore still produce an index-time estimate; live routing continues to reject
it. Direct `NewPoolSimulator` construction enables live age
checks; historical tests explicitly choose their captured evaluation time.

The current `CalcAmountOutParams` interface contains no routing-head block.
A fully block-based clock therefore needs Kyber to pass the head it already
tracks with each quote; the adapter can then compare that head with the snapshot
and posted-price blocks. Fetching or polling a head inside the adapter would add
network work to solve a context problem and is not part of this reader.

Until that caller context is available, a capture timestamp up to **1,000 ms**
ahead of the router's clock is treated as age zero. More than 1,000 ms ahead is
still `ErrStaleSnapshot`. This allowance applies only to negative ages: positive
ages, the existing one-block lookahead, on-chain maximum age, and wall-clock
backstop are unchanged. Once the router's clock passes capture time, normal
elapsed-block aging resumes immediately. It adds no RPC or polling work.

This is an explicit clock-skew tradeoff: a one-second inter-host offset can
change estimated age by up to four nominal 300 ms blocks (about 2 bp of the
stored 0.50 bp/block widening). Already-expired prices are still rejected against
the snapshot/posted-price blocks even while the clock allowance applies.

Age widening enters the existing integer curve before the solve. No fixed
short-horizon price margin is added. `GetMetaInfo` includes the snapshot block,
`quoteBlock`, and `snapshotAgeMs` at the time metadata is requested. `SwapInfo`
also carries the exact block and age used by its amount-out calculation, because
metadata requested later can cross a block-time boundary. Quote calls stay pure.

The router interface provides no execution gas price or AuctionHandler warmth.
It therefore does not invent either condition or apply an unconditional execution
haircut. `State.QuoteExecution` and `State.QuoteExactOutput` accept explicit
execution context and reproduce model 3's conditional reduction:

| Effective gas price / account context | Aged price | Price posted in execution block |
| --- | ---: | ---: |
| Gas price at most 1.5 × base fee, AuctionHandler cold | 0 bp | 0 bp |
| Gas price greater than 1.5 × base fee | 6 bp | 0.50 bp |
| Gas price greater than 2 × base fee | 8 bp | 1 bp |
| Gas price greater than 2.5 × base fee, or AuctionHandler warm | 12 bp | 2 bp |

Threshold comparisons are strict. Execution checks the full caps and preserves
the different Sell/Buy ordering of xStock dust and execution friction. A router
transaction still needs full pre-send simulation with its actual state, fee,
account accesses, approvals, deadline and minimum output.

## Complete state and pricing

Discovery reads `getPoolIds` and `getTokens`. By default the tracker runs
[`ThogSnapshotLens.sol`](abi/ThogSnapshotLens.sol) as a never-deployed constructor
through one creation `eth_call` at `latest`. The lens reads `getPoolData(0,28)`
and issuer `sharesOf(proxy)` / pool `storedMultiplier(token)` for the **listed**
xStocks in that same EVM state. Its tagged revert carries the page, share state,
block number, and block timestamp. The Go reader applies the existing schema and balance validation.
Both reader paths use nonzero simulation `gasPrice` to preserve `BASEFEE` on
clients that zero it for zero-price calls. It is unrelated to the swap gas price.

A provider that explicitly rejects creation calls or the lens gas budget uses
the Multicall3 reader; `disableLens=true` selects it directly. That reader sends
one `Multicall3.tryAggregate(false, …)` `eth_call` at `latest` to the canonical
Multicall3 (`0xcA11…CA11`). It runs `getPoolData(0,28)`, `getCurrentBlockTimestamp()`
and issuer `sharesOf(proxy)` / pool `storedMultiplier(token)` for **every**
catalog stock in one EVM state, so it is atomic like the lens. Unlisted catalog
entries may revert; the page decides which results are required. The stored
multiplier getter returns one `uint256`; issuer `getCurrentMultiplier()` is not
used. Stored multipliers are part of the pool's published pricing state and can
differ from the issuer's current multiplier.
It rejects a zero `BASEFEE` (the node ignored `gasPrice`). The returned block
timestamp feeds the same capture-time clamp as the lens.

A complete refresh gets at most **two attempts**, separated by **300 ms**. Each
attempt has a **2 s** deadline covering its entire read, including any capability
fallback. The default worst-case budget is therefore about **4.3 s**, not a
promise of continuous availability within a 15-block price lifetime. A failure
returns the original pool and an explicit error; logs include pool, dex, attempt,
and `errorKind` (`transport`, `provider_lag`, `validation`, `contract`, or `issuer`).
A lens-capability fallback is also logged explicitly. Capability matching covers
provider variants such as "user-specified gas exceeds provider limit". Transport
failures/timeouts and typed contract, issuer, or validation errors do not select
a different reader: they keep the existing whole-refresh retry policy.

Configuration fields (zero selects the default):

| Field | Default | Purpose |
| --- | ---: | --- |
| `blockTimeMs` | 300 | Nominal interval for estimated quote-block advancement |
| `maxSnapshotAgeMs` | 15000 | Wall-clock age backstop for live routing |
| `requestTimeoutMs` | 2000 | Deadline for one complete refresh attempt |
| `disableLens` | false | Select the Multicall3 reader |

The page includes all 126 logical words, native MON, physical ERC-20 balances,
accounted inventory and dynamic token/category metadata. Schema / wire / model
**8 / 8 / 3** is required. The observed registry has **17 tokens and 14 categories**;
up to 28 tokens are supported. A refresh can replace or reorder the registry.
The nine active xStocks are SPYx, QQQx, EWYx, NVDAx, TSLAx, SMHx, SPCXx, MUx, and
SKHYx. SMHx, SPCXx, MUx, and SKHYx expand the prior 13-token registry; their addresses
were already present in the catalog, so discovery needs no new hardcoded list.
The deployed catalog identifies share tokens, while the page identifies active
listings. See the integration guide for the complete address/decimals table.
Native MON contributes to risk but does not create spendable WMON output inventory.

The tracker requests another refresh after one second even without pool events;
head-driven refreshes can run sooner. `entity.Pool.Timestamp` records attempt-start
time and `BlockNumber` identifies the captured state. The millisecond capture
time and freshness configuration are stored in `Extra`.

Prices use 56-bit mantissas. Pricing includes bid/ask/category/pair spreads,
positive covariance-inventory risk, an independent quadratic input volume charge,
and integrated fixed/sliding wrapper costs. Output refunds participate in an
integer quadratic solve with an adjacent-integer check. Arithmetic uses checked
`uint256.Int` / `int256.Int`; `big.Int` is confined to ABI/RPC and router API edges.
`State.MarginalPrice` and `State.Limits` reproduce the ERC-7815 read-only surfaces.

Each quote reconciles physical holdings for its two endpoints and uses accounted
inventory for other tokens. xStock transfers floor amounts to issuer shares and
preserve the contract's dust rules. `SwapInfo` carries both next balances and next
accounted units. `UpdateBalance` applies those values without recomputing, and
clones copy all writable balance/share/word state. Quote calls do not mutate state.

Quote failures are local to a request's pair and amount: no failed-pair flag,
shared arithmetic error, or last-successful-quote cache is written. A failed
quote cannot change another pair's result. Only a deliberate `UpdateBalance`
after a selected successful fill changes portfolio state; routing alternatives
must use `CloneState` before applying those updates.

Callers should use the existing `pool.CalcAmountOut` helper, which converts a
panic in one quote into an explicit error for that call. They should continue
other pair/amount requests and should not discard an otherwise valid refreshed
pool solely because one quote fails. Whether Kyber's surrounding service disables
an entire source is a caller policy outside this adapter. Shared stale/paused or
invalid snapshot state can still make all pairs unavailable, and a required
issuer-read failure still fails the portfolio refresh: other pairs can depend on
that issuer state through portfolio risk. Pair isolation does not omit that data.

Gas estimates are 273,846 within a category and 564,978 across categories. They are
the integer averages of the `swapWithExtraFees` frame gas in Tenderly simulations of
router transactions on Monad (3 and 14 swaps; the ranges were 270,713-275,413 and
538,985-597,354: xStock output costs about 597,000, xStock input about 539,000).
The model-3 execution harness measures up to 202,019 and 397,888 with some state
pre-warmed, so tests check those against the estimates. Remeasure after changes to
token code, the registry or policy.

## Local quote performance

The quote path extracts narrow packed fields directly from their uint256 limbs,
reuses unchanged token valuations when checking category inventory, skips zero
covariance contributions, and handles unit share multipliers / common dust values
with exact identities. No mutable prepared-state cache is added.

Against `a6db4e4a3e4edb36ca4e74a4408faf5cbbe23067`, the current-registry benchmarks
measured the following medians on Apple M1 Max, darwin/arm64, Go 1.27.2, one CPU.
Twelve old/new pairs alternated execution order with a one-second benchmark time;
both binaries used the same block-122916130 fixture, inputs and quote block.

| Pair | Before | After | Median reduction | benchstat p-value |
| --- | ---: | ---: | ---: | ---: |
| USDC → WETH | 6.945 µs | 6.199 µs | 10.7% | 0.052 |
| WETH → USDC | 6.514 µs | 5.960 µs | 8.5% | 0.033 |
| USDC → SMHx | 7.183 µs | 5.270 µs | 26.6% | <0.001 |
| MUx → USDC | 6.274 µs | 5.241 µs | 16.5% | 0.014 |
| MUx → SKHYx | 7.005 µs | 6.355 µs | 9.3% | 0.114 |

The geometric mean decreased 14.6%. USDC → WETH and MUx → SKHYx are not
statistically significant at 5% in this run, so their medians are descriptive.
Every quote remains **368 B / 7 allocations**. These measurements cover local
simulation CPU, not RPC, transaction inclusion, or production end-to-end latency.
[Before](testdata/bench-current-before.txt) and
[after](testdata/bench-current-after.txt) samples are included for `benchstat`.

```sh
GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -run '^$' \
  -bench '^BenchmarkCurrentRegistry$' -benchmem -cpu 1 -benchtime 1s -count 12
benchstat pkg/liquidity-source/thog-prop/testdata/bench-current-{before,after}.txt
```

## Verification

The full published corpus passes 61,920 contract comparisons after the reader and
math changes: 29,397 exact successes and 32,523 matching rejections. The committed
fixtures are selected observations with unchanged expected outputs:

| Corpus | Full cases | Exact successes | Matching rejections | Committed cases |
| --- | ---: | ---: | ---: | ---: |
| Historical schema-8 read-only snapshot | 1,872 | 954 | 918 | 96 |
| Historical curve/share/cap/age boundaries | 24,024 | 9,982 | 14,042 | 1,232 |
| Model-3 local-fork execution contexts | 32,760 | 16,428 | 16,332 | 1,692 |
| Current 17-token stored-multiplier snapshot, block 122916130 | 3,264 | 2,033 | 1,231 | 312 |
| **Total** | **61,920** | **29,397** | **32,523** | **3,332** |

The current full capture covers all 272 directed token pairs. Its excerpt includes
both USDC directions for all nine stocks, new-stock crosses, and ordinary-token
controls. Its implementation runtime SHA-256 is
`b74a0429501ae2dd11695b67a74243c6b4554f254c924b129b87327bb8c30846`.

Historical model-2 read-only fixtures remain useful because model 3 changes the
conditional execution charge, not the read-only curve. The test helper changes
only their identity byte to model 3; runtime snapshots never receive that change.
The execution corpus uses a locally compiled model-3 implementation on a Monad
fork. Additional committed regression fixtures retain 96 older deployed-model-3
cases and 320 one-block/age-cutoff cases. The latter tests $1 and $100 in every
then-live pair plus maximum-age/first-stale-block cases; only the evaluation block
changes, and the router's spread increment must equal 50.

`TestVerification` (manual; needs `THOG_PROP_RPC_URL` and
`THOG_PROP_TENDERLY_{ACCOUNT,PROJECT,ACCESS_KEY}`) checks the live pool end to end:
the real tracker reads head N, the simulator quotes at N+1, and each of 40 cases
(8 pairs × $1–$50M) settles as a Tenderly bundle on N's state with N+1's header
(Tenderly's `block_number` alone runs on the state after that block). On November 3
it passed in 13 s: 25 exact, 11 matching rejections, and 4 SPYx outputs 1 wei above
the guarantee from share rounding. `TestVerificationMultiHop` settles 21 routes that
hit the pool two or three times in one bundle (round trips, repeats, chains through
SPYx); every later hop is quoted on a clone after `UpdateBalance` and all 48 hops
matched (42 exact, 6 SPYx +1 wei). `TestCloneStateConcurrentRoutesAreIsolated` replays
such routes on 16 clones in parallel under `-race`.

The live stored-multiplier lens capture at block **122921791** verifies all 17
listed tokens and nine share holdings. Read-only token-code overrides keep
`sharesOf` identical and make every other issuer selector revert. The updated
lens returns the identical snapshot; the prior lens fails at
`getCurrentMultiplier()`. The capture records both responses, code hashes, and
overrides. The older lens fixture remains for the historical 32-case age-cutoff
test (30 exact outputs and two stale rejections).

Both real Go readers also returned identical state and timestamp at block
**122924036**, pinned by canonical block hash. This manual check is reproducible:

```sh
THOG_PROP_READER_RPC_URL="$MONAD_RPC" GOTOOLCHAIN=go1.27.2 go test \
  ./pkg/liquidity-source/thog-prop -run '^TestLiveSnapshotReadersAgree$' -v
python3 pkg/liquidity-source/thog-prop/testdata/build_lens.py --solc /path/to/solc-0.8.25 --check
```

Additional tests cover fee thresholds at odd/zero/max-uint256 base fees, snapshot
validation, the Multicall3 reader (one atomic `latest` call, exact multiplier
getter target/argument/return shape, typed failures), catalog equality across Go
and the lens, refreshes without events, quote purity, clone isolation,
share/accounted inventory updates, and Msgpack serialization. Packed-field tests
compare every valid width/offset with a reference integer mask; share conversion
tests cover unit multipliers, dust thresholds, and rounding boundaries.

The [fixture notes](testdata/README.md) describe provenance and reproduction.
Fixtures use a nested module and `os.ReadFile`, so downstream module downloads
exclude them. Capture scripts do not submit transactions to mainnet.

```sh
GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -race
GOTOOLCHAIN=go1.27.2 go vet ./pkg/liquidity-source/thog-prop
GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -run '^$' -bench . -benchmem
# Optional: use complete captures instead of the selected excerpts.
THOGAMM_FIXTURES=/path/to/thog-amm-integration-guide/fixtures GOTOOLCHAIN=go1.27.2 go test \
  ./pkg/liquidity-source/thog-prop -v \
  -run '^(TestHistoricalReadOnlyContractParity|TestHistoricalBoundaryParity|TestModel3ExecutionParity|TestStoredMultiplierSeventeenTokenParity)$'
```

## Refreshing an existing integration

Rebuild the adapter and refresh stored pool rows after the upgrade so their share
multipliers and registry come from the current pool. The proxy, pool ID, schema,
wire format, model version, quote APIs, and configuration are unchanged. Maximum
price age and widening continue to come from the published on-chain control word.
The local quote optimizations do not change freshness checks, retries, fees, caps,
rounding, or quote rejection rules.

The read-only provider probe accepts explicitly supplied endpoints:

```sh
python3 pkg/liquidity-source/thog-prop/testdata/refresh_probe.py \
  --rpc "$MONAD_RPC" --count 20 --output /tmp/thog-refresh-probe.json
```

Integration requests for Kyber:

- Pass the already-known routing-head block with quote context so aging can use
  block numbers without adapter RPCs. This needs caller/interface wiring; the
  current simulator interface carries only the stored snapshot block.
- Keep the one-second schedule and confirm whether successful `MakerPageUpdated`
  publications can trigger refreshes sooner. The event indicates a publication;
  it is not a guarantee that the maker publishes in every block.
- Confirm that the executor's minimum output uses the net amount consistent with
  the encoded integrator fee and intended slippage tolerance. The curve already
  includes modeled age widening; do not automatically grant it twice as an extra
  execution tolerance. Executor changes are outside this dex-lib adapter.

The guide and polling/capture SDK's hash-pinning guidance require follow-up in
their own repositories. No takerbot restart or production configuration change
is part of this PR. The October 28 rally replay and any fixed quote margin are
outside this change.
