# ThogAMM schema-8 / model-3 adapter

ThogAMM is a portfolio-inventory AMM on Monad (chain ID 143). Integrate the proxy
`0x80c74517bcc2d67ffe02d3ed886796272f647210`. The implementation observed at block
117211724 is `0xd84cb12d870344195f23c0a5faf70e3221bdbd9e`. Pool ID:
`0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc`.

[Integration guide and integer specification](https://github.com/thogard785/thog-amm-integration-guide/blob/main/THOGAMM_AGGREGATOR_INTEGRATION.md)
· [Proxy on Monadscan](https://monadscan.com/address/0x80c74517bcc2d67ffe02d3ed886796272f647210)
· [Implementation](https://monadscan.com/address/0xd84cb12d870344195f23c0a5faf70e3221bdbd9e)
· [Reference Rust model](https://github.com/thogard785/thog-amm-poll-sdk/tree/v0.5.0/model)

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
and `sharesOf` / `getCurrentMultiplier` for the **listed** xStocks in that same
EVM state. Its tagged revert carries the page, issuer state, block number, and
block timestamp. The Go reader applies the existing schema and balance validation.
Both reader paths use nonzero simulation `gasPrice` to preserve `BASEFEE` on
clients that zero it for zero-price calls. It is unrelated to the swap gas price.

A provider that explicitly rejects creation calls or the lens gas budget uses
the page-first reader; `disableLens=true` selects it directly. That reader takes
the block number from `getPoolData` itself and number-pins every issuer call.
It also confirms the page at that number in the issuer batch to detect a changed
page/reorg. There is no independent header request or EIP-1898 hash parameter.
An issuer read is never retried at `latest` alongside an older page. The lens is
atomic by construction; the fallback uses number pinning and consistency checks
and does not claim hash-canonical pinning across load-balanced nodes.

A complete refresh gets at most **two attempts**, separated by **300 ms**. Each
attempt has a **2 s** deadline covering its entire read, including any capability
fallback. The default worst-case budget is therefore about **4.3 s**, not a
promise of continuous availability within a 15-block price lifetime. A failure
returns the original pool and an explicit error; logs include pool, dex, attempt,
and `errorKind` (`transport`, `provider_lag`, `reorg`, `validation`, `contract`, or `issuer`).
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
| `disableLens` | false | Select the page-first number-pinned reader |

The page includes all 126 logical words, native MON, physical ERC-20 balances,
accounted inventory and dynamic token/category metadata. Schema / wire / model
**8 / 8 / 3** is required. The observed registry has 13 tokens and 10 categories;
up to 28 tokens are supported. A refresh can replace or reorder the registry.
The deployed catalog identifies which tokens use issuer shares, while the page
identifies active listings. Native MON contributes to risk but does not create
spendable WMON output inventory.

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

Gas estimates are 273,846 within a category and 564,978 across categories. They are
the integer averages of the `swapWithExtraFees` frame gas in Tenderly simulations of
router transactions on Monad (3 and 14 swaps; the ranges were 270,713-275,413 and
538,985-597,354: xStock output costs about 597,000, xStock input about 539,000).
The model-3 execution harness measures up to 202,019 and 397,888 with some state
pre-warmed, so tests check those against the estimates. Remeasure after changes to
token code, the registry or policy.

## Verification

The full external corpus was checked before selecting the committed excerpts:

| Corpus | Full cases | Exact successes | Matching rejections | Committed cases |
| --- | ---: | ---: | ---: | ---: |
| Historical schema-8 read-only snapshot | 1,872 | 954 | 918 | 96 |
| Historical curve/share/cap/age boundaries | 24,024 | 9,982 | 14,042 | 1,232 |
| Model-3 local-fork execution contexts | 32,760 | 16,428 | 16,332 | 1,692 |
| Deployed model-3 snapshot, block 117211724 | 1,872 | 1,121 | 751 | 96 |
| Deployed model-3 one-block and age-cutoff cases | 320 | 291 | 29 | 320 |
| **Total** | **60,848** | **28,776** | **32,072** | **3,436** |

Historical model-2 read-only fixtures remain useful because model 3 changes the
conditional execution charge, not the read-only curve. The test helper changes
only their identity byte to model 3; runtime snapshots never receive that change.
The execution corpus used a locally compiled model-3 implementation on a Monad
fork before deployment. The two new model-3 read-only corpora use the actual
deployed implementation bytecode, whose SHA-256 is
`92d7605501570cc4871368ad54f764215d0016c6eb159ad3fd7aa0598a657449`.

The one-block corpus tests both $1 and $100 in every directed live pair, plus
maximum-age/first-stale-block cases. Anvil overrides only the evaluation block
number while keeping the captured state fixed. The router is compared directly
with those EVM outputs and its spread increment is checked to equal 50.

A live atomic-lens capture from October 29 is committed as `testdata/lens-monad.json.gz`.
It verifies decoding the compiled lens result, nonzero base fee, all 13 listed
tokens, and all five listed issuer states. A second 32-case corpus uses the same
deployed bytecode on an isolated Monad fork with only `CONTROL.maxAge=15` and the
evaluation block overridden. It covers every modeled execution age through the
cutoff for USDC→WMON and USDC→xStock: 30 exact outputs and two stale rejections.
The compiled bytecode is reproducible:

```sh
python3 pkg/liquidity-source/thog-prop/testdata/build_lens.py --solc /path/to/solc-0.8.25 --check
```

Additional tests cover fee thresholds at odd/zero/max-uint256 base fees, snapshot
validation, number-pinned reads and reorg rejection, reordered issuer replies, refreshes
without events, quote purity, clone isolation, share/accounted inventory updates,
and Msgpack serialization. A live tracker → JSON → simulator smoke test passed at
block 117214675 with all 13 tokens.

The [fixture notes](testdata/README.md) describe provenance and reproduction.
Fixtures use a nested module and `os.ReadFile`, so downstream module downloads
exclude them. Capture scripts do not submit transactions to mainnet.

```sh
GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -race
GOTOOLCHAIN=go1.27.2 go vet ./pkg/liquidity-source/thog-prop
GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -run '^$' -bench . -benchmem
# Optional: use complete captures instead of the selected excerpts.
THOGAMM_FIXTURES=/path/to/full/fixtures GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -v
```

## Provider probe and rollout

The local October 29 probe ran six successful reads per design on
`rpc.monad.xyz`: median complete-read latency was **149 ms** with a header/hash,
**136 ms** with a header/number, **122 ms** page-first, and **61 ms** through the
lens. These are small samples without retries, not reliability estimates.

A subsequent independent review supplied these additional live lens checks:

| Endpoint | Successful lens checks | Reported latency |
| --- | --- | --- |
| `rpc.monad.xyz` | Success; page matched a same-block direct read, five issuers decoded | 72 ms |
| `rpc1.monad.xyz` | 3/3 | Not reported |
| `rpc-mainnet.monadinfra.com` | 3/3 | Not reported |
| Ankr | 3/3 | Not reported |
| dRPC | 3/3 | Not reported |

The reviewer supplied these results separately from the local probe. Earlier
HTTP 403s in the local environment did not test RPC support; the subsequent
successful checks establish lens compatibility on those providers. Kyber can
run the committed probe against its own provider URLs:

```sh
python3 pkg/liquidity-source/thog-prop/testdata/refresh_probe.py \
  --rpc https://rpc.monad.xyz --count 20 --output /tmp/thog-refresh-probe.json
```

Before re-enabling the source, verify that the **on-chain** control word contains
the intended maximum age of 15. Updating takerbot's YAML alone does not establish
that it has published that policy. Then rebuild/refresh stored pool state with
this adapter. The captured October 29 lens fixture still has a maximum age of 5;
the adapter must honor it. After rollout, inspect `ErrStaleSnapshot` and classified
refresh-error rates, as well as the served snapshot/quote blocks.

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
