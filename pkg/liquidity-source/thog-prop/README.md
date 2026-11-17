# ThogAMM adapter (`thog-prop`)

ThogAMM is a market-maker pool on Monad (chain ID 143). One contract holds one
pool of up to 28 ERC-20 tokens and swaps any listed token for any other from its
own inventory. This adapter reads the pool's complete state in one `eth_call` and
reproduces the contract's read-only quote in integer arithmetic.

- Pool contract (a proxy; send every call here):
  [`0x80c74517bcc2d67ffe02d3ed886796272f647210`](https://monadscan.com/address/0x80c74517bcc2d67ffe02d3ed886796272f647210)
- Pool ID: `0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc`
- [Integration guide](https://github.com/thogard785/thog-amm-integration-guide/blob/main/THOGAMM_AGGREGATOR_INTEGRATION.md)
  and [exact arithmetic and state layout](https://github.com/thogard785/thog-amm-integration-guide/blob/main/REFERENCE.md)

## How the pool prices a swap

The market maker publishes a USD price for every token. A swap values the input
at its price, subtracts charges, and converts the rest into the output token.
The charges are a spread (per token, per category of tokens that track one
asset, per pair, plus a widening for every block since the last price update), a
charge when the swap adds risk to the pool's whole inventory, a charge that grows
with trade size, and an optional per-token cost. A quote for one pair therefore
depends on the balances of every token. Inventory caps and a maximum price age
limit what can be quoted.

Listed xStocks are **share-based**: the token contract records balances as shares
and converts them to token amounts with a multiplier, so a transfer moves a whole
number of shares and can deliver a few raw units less than requested. The pool
keeps its own copy of each multiplier (`storedMultiplier(token)`) and uses it for
pricing and settlement. `shares.go` reproduces the rounding; the guide links the
xStocks documentation.

## Reading state

Discovery calls `getPoolIds` and `getTokens`. Each refresh reads, in one EVM
state at `latest`:

1. `getPoolData(0, 28)`: the packed pricing words, every token balance, and the
   block number, base fee and parent hash. [`layout.go`](layout.go) names each field.
2. `storedMultiplier(token)` on the pool for every listed token. The pool stores
   a multiplier only for share-based tokens; **zero means an ordinary ERC-20**.
   No token address is compiled into the adapter.
3. `sharesOf(pool)` on each token whose multiplier is nonzero.

The default reader is [`ThogSnapshotLens.sol`](abi/ThogSnapshotLens.sol). It is
never deployed: its creation code runs as one `eth_call` and returns the result
as revert data. A provider that refuses creation calls, or the lens's 3,000,000
gas limit, is read through one `Multicall3.tryAggregate` call instead, and the
tracker remembers the refusal for ten minutes. `disableLens` selects Multicall3
directly. Its batch is built from the pool's token list; if the pool lists
different tokens than the batch asked about, the tracker reads once more with
the pool's own list and never combines two reads.

Both readers send `gasPrice: 0x1`: Monad nodes report a zero `BASEFEE` to a call
without a gas price, and the snapshot must carry the real one. A refresh makes at
most two attempts 300 ms apart, each with a 2 s deadline. A failure returns the
unchanged pool and an error logged with `errorKind` (`transport`, `provider_lag`,
`validation`, `contract` or `issuer`). The next refresh is due after one second.

## Quote freshness

Prices age by block, so the simulator prices at the block a swap would execute in:

```text
elapsed blocks = ceil((now - capturedAt) / blockTime)
quote block    = snapshot block + elapsed blocks + 1
age            = quote block - block of the last price update
spread        += age × the pool's widening per block
```

`capturedAt` is the start of the successful read, clamped to the end of the
block's timestamp second so a lagging provider cannot make old state look new.
The factory honours `FactoryParams.Opts.StaleCheck`:

- **`true` (routing)**: a quote fails with `ErrStaleSnapshot` when the age exceeds
  the pool's maximum age, when the snapshot is older than `maxSnapshotAgeMs`, or
  when its capture time is missing or more than one second ahead of the local clock.
- **`false` (indexing)**: age never rejects a quote. The simulator prices at
  snapshot block + 1 and keeps every other check.

## What a quote covers

`CalcAmountOut` equals the contract's `makerQuoteExactInput` at the quote block,
including share rounding and the caps a read-only quote applies (90% of each
inventory cap). `SwapInfo` carries the post-swap balances; `UpdateBalance`
applies them without recomputing and `CloneState` copies all mutable state, so a
route can visit the pool more than once. A failed quote changes nothing and
leaves the pool's other pairs quotable.

Two settlement adjustments are outside `CalcAmountOut`:

- **Execution surcharge.** At settlement the pool reduces output when the
  transaction's effective gas price is high relative to the block base fee, or
  when the transaction has already touched the FastLane AuctionHandler. The
  router interface carries neither. `State.QuoteExecution` and
  `State.QuoteExactOutput` take them as an `ExecutionContext` and apply the
  contract's rule (`friction` in the code):

  | Effective gas price (first match) | Surcharge | Prices updated in the same block |
  | --- | ---: | ---: |
  | AuctionHandler warm, or above 2.5 × base fee | 12 bp | 2 bp |
  | above 2 × base fee | 8 bp | 1 bp |
  | above 1.5 × base fee | 6 bp | 0.5 bp |
  | otherwise | 0 | 0 |

- **Integrator fee.** `swapWithExtraFees` deducts
  `floor(output × extraFeePpm / 1,000,000)` before its minimum-output check.
  `extraFeePpm` is in parts per million (100 is 1 bp). The adapter quotes the
  output before this fee.

`CalcAmountOut` reports 273,846 gas within one category and 564,978 across
categories: averages of the `swapWithExtraFees` frame in simulated router
transactions on Monad (ranges 270,713–275,413 and 538,985–597,354).

## Configuration

`proxy` and `poolId` are required. A zero selects the default for the rest:
`blockTimeMs` 300 (block interval used to estimate the quote block),
`maxSnapshotAgeMs` 15000 (oldest snapshot a routing quote may use),
`requestTimeoutMs` 2000 (deadline of one refresh attempt), and `disableLens`
false (read through Multicall3 only).

## Tests

```sh
GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -race
GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -run '^$' -bench . -benchmem
```

Quote tests compare the simulator with results recorded from the contract: exact
outputs for accepted requests, and agreement on which requests are rejected. The
committed fixtures are excerpts, described in [testdata/README.md](testdata/README.md).
Further checks need the complete fixtures, a live endpoint or solc; none sends a
transaction.

```sh
# All 61,920 published cases.
THOGAMM_FIXTURES=/path/to/thog-amm-integration-guide/fixtures GOTOOLCHAIN=go1.27.2 go test \
  ./pkg/liquidity-source/thog-prop -v \
  -run '^(TestHistoricalReadOnlyContractParity|TestHistoricalBoundaryParity|TestModel3ExecutionParity|TestStoredMultiplierSeventeenTokenParity)$'

# Both readers return the same state at one block.
THOG_PROP_READER_RPC_URL="$MONAD_RPC" GOTOOLCHAIN=go1.27.2 go test \
  ./pkg/liquidity-source/thog-prop -run '^TestLiveSnapshotReadersAgree$' -v

# Quotes against simulated settlement, single- and multi-hop. Needs
# THOG_PROP_RPC_URL and THOG_PROP_TENDERLY_{ACCOUNT,PROJECT,ACCESS_KEY}.
GOTOOLCHAIN=go1.27.2 go test ./pkg/liquidity-source/thog-prop -run '^TestVerification' -v

# The committed lens bytecode matches its source.
python3 pkg/liquidity-source/thog-prop/testdata/build_lens.py --solc /path/to/solc-0.8.25 --check
```
