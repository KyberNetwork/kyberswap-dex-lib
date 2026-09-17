# INVERSE routed hook (draft, no active registration)

This package implements exact-input quotes for the fee-aware `RoutedInverseHook`
candidate from [inversecoin](https://github.com/calmdentist/inversecoin/tree/ec44bc670be90057188fcd6d02f14848e6010163).
**This candidate is not deployed.** `HookAddresses` is intentionally empty.
Do not register the older live hook `0x2f293d502485Ef363A959140C3CebEE415902aec`:
its protocol-fee and settlement rules differ from this candidate.

The integration reuses v4 pool discovery, tracking and execution metadata. There
is no new on-chain adapter, arbitrary hook data, token wrapper or RFQ dependency.
It provides a local pricing model so the quote service can understand the custom
swap hook instead of treating its resident native liquidity as an ordinary AMM.

## Pricing and settlement

Ownership uses fixed shares. With reference reserves `x` (shares), `y` (WETH),
initial reserves `x0,y0`, and `R=10^27`, the next denomination is

```
relative = floor(y' * x0 * R / (x' * y0))
index'   = floor(relative * relative / R)
```

Buys calculate constant-product share output, then return
`floor(sharesOut * index' / R)` nominal token units. Sells convert input using
`ceil(nominalIn * R / currentIndex)` shares, then calculate WETH output. The fee
is `protocol + 3000 - floor(protocol * 3000 / 1_000_000)`, with the direction's
packed protocol fee. Protocol reserves round up exactly as the candidate does.
Native v4 protocol fees are rounded separately at each bitmap-word step.

The hook removes its sole full-range position, changes the denomination, rebuilds
the position and executes a native swap before reconciling the economic output.
Consequently the model also checks deposit custody, actual fee collection,
per-removal WETH loss (maximum 32 wei), remaining rounding funds, native output
correction and the endpoint price tolerance. A formula-only quote is insufficient.

`BeforeSwap` consumes all input in the off-chain dispatcher and supplies the
complete output: the ordinary curve must not price the trade again. All state is
value-owned; `UpdateBalance` applies the saved transition without recalculating.
The optional `HookPoolStateProvider` refreshes v4 reserves, sqrt price, liquidity
and tick after a rebase. Tick ranges remain the two fixed full-range boundaries.

FullMath intermediates use `big.Int` with explicit result bounds; snapshot and
native swap primitives use `uint256.Int`. Math helpers do not mutate arguments.
There are no floating point price calculations or RPC calls during quoting.

## State and supported configuration

- Robinhood Chain 4663; trusted, non-rebasing WETH only, 18 decimals on both sides.
- PoolManager `0x8366a39cc670b4001a1121b8f6a443a643e40951`.
- WETH `0x0bd7d308f8e1639fab988df18a8011f41eacad73`.
- StateView `0xf3334192d15450cdd385c8b70e03f9a6bd9e673b` (configurable in v4).
- Static LP fee 3000, tick spacing 60, sole hook-owned position [-887220,887220].
- Directional protocol fee 0..1000 pips. Fee-bearing INVERSE collection is enabled
  only for the reviewed permissionless Robinhood controller
  `0x6d0009504d129cf5002dba61d9ae8575aa79314c`. An unknown replacement fails closed
  for sells requiring collection; buys/zero-collection sells remain quotable.
- Idle snapshots only: no prepayment/settlement in flight, no uncollected INVERSE
  protocol fees, coherent token/hook/manager identity and index, positive backing.
- Closed pools return zero reserves; malformed/unsupported snapshots cannot quote.

Two dependent multicall batches are pinned to the v4 tracker's exact block and
both receive state overrides. Position fee growth uses modulo-2^256 subtraction.
The resolved block is retained in `entity.Pool` and standard execution metadata.
Hook immutables are included in the versioned hook snapshot because the shared v4
StaticExtra does not contain hook-specific fields; they are checked every refresh.

## Execution policy required in the quote service

Initially enable **one INVERSE leg per transaction**. Use exact input, empty hook
data, full atomic INVERSE settlement/take, and ordinary v4 extreme price limits.
Prepaid and postpaid settlement are supported by the candidate. Exact output,
partial INVERSE takes, persistent INVERSE ERC-6909 credits and external LP
positions are unsupported. Wallet balances/allowances and slippage still need
normal transaction simulation.

Rebasing changes nominal holdings across legs. The simulator updates this pool's
own state correctly, but the public DexLib cannot enforce a backend's split-route
or cycle policy or re-denominate balances in unrelated pools. Do not activate
unrestricted split/cyclic routes using this token without provider-side handling
and execution tests. This is a release gate, not a claim that a draft PR enables
Kyber's production API or Fomo.

The 900,000 hook-gas allowance is provisional and additive to the v4 dispatcher's
base gas. Calibrate it against deployed adapter receipts before activation.

## Tests and activation

See [testdata/README.md](testdata/README.md) for pinned sources and reproduction.

```
CI=true go test -race ./pkg/liquidity-source/uniswap/v4/... ./pkg/msgpack/...
go test ./pkg/liquidity-source/uniswap/v4/hooks/inverse -run '^$' \
  -fuzz FuzzQuoteDoesNotMutate -fuzztime=30s
```

The checked-in cases come from actual local v4-core + candidate execution, not a
copy of the Go formula. They compare output and **every modeled post-swap field**,
including LP fees, custody, native price/liquidity, index and rounding buffer.
Tests also cover tracking failures, overrides, clone isolation, no mutation on
quote failure, sequential v4 dispatch and msgpack round trips.

Before enabling registration: publish/pin candidate source; deploy and verify its
bytecode and immutable arguments; register the exact hook address; capture a
pinned Robinhood snapshot and successful buy/sell execution through the deployed
Kyber adapter; verify routing policy and gas; then deploy the quote-service update.
No deployment, liquidity withdrawal or funded swap is part of this PR.
