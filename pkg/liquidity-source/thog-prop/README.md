# ThogAMM schema-8 / model-3 adapter

ThogAMM is a portfolio-inventory AMM on Monad (chain ID 143). Integrate the proxy
`0x80c74517bcc2d67ffe02d3ed886796272f647210`. The implementation observed at block
117211724 is `0xd84cb12d870344195f23c0a5faf70e3221bdbd9e`. Pool ID:
`0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc`.

[Integration guide and integer specification](https://github.com/thogard785/thog-amm-integration-guide/blob/main/THOGAMM_AGGREGATOR_INTEGRATION.md)
· [Proxy on Monadscan](https://monadscan.com/address/0x80c74517bcc2d67ffe02d3ed886796272f647210)
· [Implementation](https://monadscan.com/address/0xd84cb12d870344195f23c0a5faf70e3221bdbd9e)
· [Reference Rust model](https://github.com/thogard785/thog-amm-poll-sdk/tree/v0.5.0/model)

## One-block quote policy

`CalcAmountOut` evaluates the read-only exact-input curve at **snapshot block + 1**,
with unchanged valuations and inventory. This adds one block of the contract's
configured age widening: **50 hundredths of a basis point = 0.50 bp** at the
captured live policy. Existing price age is retained:

```text
quote age = snapshot block + 1 - posted price block
age spread = quote age × on-chain widening per block
```

The additional 0.50 bp enters the curve's spread before its integer solve; it is
not a post-quote multiplication. The contract's maximum age and strict 90%
read-only receiving caps still apply at that quote block. `GetMetaInfo` returns
the actual snapshot block, so callers can identify the source state.

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

Discovery reads `getPoolIds` and `getTokens`. The tracker reads a canonical header,
then pins `getPoolData(0,28)` and batched issuer `sharesOf`/`getCurrentMultiplier`
calls to that same hash with `requireCanonical: true`. It validates the page's
block number, parent hash and base fee against the header. Missing issuer data,
inconsistent balances or a reorg fail the refresh explicitly.

The page includes all 126 logical words, native MON, physical ERC-20 balances,
accounted inventory and dynamic token/category metadata. Schema / wire / model
**8 / 8 / 3** is required. The observed registry has 13 tokens and 10 categories;
up to 28 tokens are supported. A refresh can replace or reorder the registry.
The deployed catalog identifies which tokens use issuer shares, while the page
identifies active listings. Native MON contributes to risk but does not create
spendable WMON output inventory.

The tracker requests another refresh after one second even without pool events;
head-driven refreshes can run sooner. `entity.Pool.Timestamp` records receipt
time and `BlockNumber` identifies the captured state. The simulator uses block
age and never extrapolates blocks from elapsed wall time.

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

Gas estimates are 350,000 within a category and 550,000 across categories. They
retain allowance for cold-call overhead: the earlier cold-call study measured
222,072–323,820 and 343,043–521,306 respectively. The current model-3 execution
harness measures up to 202,019 and 397,888, which tests check against the estimates.
The harness warms some state while preparing cases, so these newer measurements
are not a replacement for a complete cold router transaction estimate. Both
studies use Monad Foundry 1.8.6 with page-storage pricing. Remeasure after changes
to token code, the registry or policy.

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

Additional tests cover fee thresholds at odd/zero/max-uint256 base fees, snapshot
validation, canonical RPC pinning, missing/reordered issuer replies, refreshes
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
