# EVPLUSAI (Robinhood Chain)

This handler integrates the immutable native ETH/USDG dynamic-fee hook. It is
off-chain quoting code; no new hook deployment or custom swap calldata is needed.

## Contracts and pool discovery

- Chain ID: **4663**.
- Hook: `0xcB787A5cDEA8B3715d984d82F1203Fd7bFeBE0c4`.
- PoolManager: `0x8366a39cc670b4001a1121b8f6a443a643e40951`.
- StateView: `0xf3334192d15450cdd385c8b70e03f9a6bd9e673b`.
- Quoter: `0x8dc178efb8111bb0973dd9d722ebeff267c98f94`.
- Pool ID: `0xc7b615a3721594f73664eb3f62d8290d0fcc8d9d1156aed1ddecbd7f32efd9f5`.
- Pool key: native ETH (`address(0)`), USDG
  (`0x5fc5360d0400a0fd4f2af552add042d716f1d168`), dynamic fee `8388608`,
  tick spacing **10**, hook above.
- Source: https://robinhoodchain.blockscout.com/address/0xcB787A5cDEA8B3715d984d82F1203Fd7bFeBE0c4?tab=contract
- Source match: https://sourcify.dev/server/v2/contract/4663/0xcB787A5cDEA8B3715d984d82F1203Fd7bFeBE0c4
- Metadata registration: https://github.com/Uniswap/hooklist/pull/10432

Reuse the parent v4 PoolManager Initialize-event/StateView discovery path. This
package recognizes the exact deployed hook address; pool service must onboard the
pool separately. One hook may initialize multiple approved pools; state is keyed
by pool ID. Future pools using this same deployment do not require a new address
entry. Redeployed hooks require review and registration.

Native ETH is represented by wrapped-native address
`0x0bd7d308f8e1639fab988df18a8011f41eacad73` in entity tokens and by zero address in
the execution pool key. The standard v4 metadata path handles that distinction.

## Fees

The worker posts two nominal combined budgets F, between 250 and 9000 pips
(0.025%-0.9%). The immutable nominal allocation is 90% LP / 10% platform, subject
to integer rounding. These are final budgets, not additions to the base fee.

- Exact input: `L = floor(9F/10)`, platform charge
  `floor(actualCoreOutput * F / (10 * (1_000_000 - L)))`.
- Exact output: `L = floor(9F*1_000_000/(10_000_000-F))`, platform charge
  `floor(actualCoreInput * F / (10_000_000-F))`.

beforeSwap overrides the LP rate L. Core protocol fees remain additional and
directional: `P + L - floor(P*L/1_000_000)`. The parent simulator expects this
combined core rate as its override. afterSwap subtracts the platform charge from
exact-input output, or adds it to exact-output input. `currentFee` must not be
used directly as the core LP rate. There is no second platform deduction at LP
withdrawal/collection.

## Tracking and freshness requirements

Track batches `registered`, `feeStateOf`, `getSlot0` and the chain timestamp at
the parent's pinned block. It propagates state overrides. Unregistered pools,
invalid snapshots and wrong-chain configuration fail closed. At
`timestamp >= expiresAt`, pricing uses the 500-pip base fee (0.05%).

The simulator is deterministic at its snapshot block; it does not read the
machine's clock between beforeSwap and afterSwap. **Pool service must refresh
on FeesPosted, WorkerSet/FeesReset, and expiry, even when no Swap event occurs.**
Tracking every new block is one possible scheduling policy. ExpiresAt is exposed
in HookExtra for scheduling. Production onboarding must configure this refresh
policy; adding an adapter cannot independently configure the backend scheduler.
Subsequent worker updates or market movements can invalidate any prior quote;
normal router slippage limits and transaction simulation still apply.

## Exact rounding and serialization

The shared v3 core normally approximates rounding while jumping to initialized
ticks. A deployed Quoter comparison exposed a one-unit USDG underquote on a
0.1 ETH exact-output buy at a 9000-pip budget. This handler opts into exact bitmap
word traversal on a quote-local core copy. Other handlers retain their existing
mode. The transient mode flag is excluded from MessagePack; it is re-enabled
from the hook on each quote and does not change the persisted core field layout.

Hook registration is generated in pkg/msgpack. The base hook is embedded as a
concrete value. A full simulator roundtrip also required registration of the
existing Few TokenWrapper value stored behind the parent v4 wrapper interface;
the same missing registration reproduced on a no-hook v4 simulator.

## Verification

`testdata/quotes.json` contains 40 exact comparisons at Robinhood block
**68585551**, with actual tick/liquidity state and the deployed Quoter's results.
It covers buys/sells, exact input/output, two sizes, live state, minimum/maximum
directional budgets, odd pip values, and the exact expiry boundary. Fee scenarios
use read-only eth_call state overrides of the verified `_fees` mapping (slot 5).
No transaction or private key is involved. The larger buys cross a bitmap word.

Offline tests replay all 40 expected outputs, check repeated/clone quotes,
directional protocol fee composition, zero/tiny charges, invalid snapshots and
amounts, native execution metadata, and full simulator MessagePack roundtrip.

Run:

```sh
CI=true go test ./pkg/liquidity-source/uniswap/v4/hooks/evplusai ./pkg/liquidity-source/uniswap/v3 ./pkg/liquidity-source/uniswap/v4
EVPLUSAI_RPC_URL=https://rpc.mainnet.chain.robinhood.com EVPLUSAI_BLOCK=68585551 go test ./pkg/liquidity-source/uniswap/v4/hooks/evplusai -run TestRPCQuotes -v
```

To regenerate the fixture after successful comparisons, additionally set
`EVPLUSAI_WRITE_FIXTURE=testdata/quotes.json`. Keep the pinned block for reproducibility.

Callback gas allowances are 15,000 beforeSwap and 45,000 afterSwap, added to the
parent's base/tick gas. A local Anvil fork at block 68585551, with one locally
mined block to initialize its EVM header, measured 6,382-6,472 and 14,377-14,544
gas respectively across four swap kinds at fallback fees with existing claims.
The allowances include room for cold access and first-claim storage writes;
these are estimates, not an exact gas guarantee. Quoter swap gas in the pinned
fixtures is approximately 62,899-68,914; it is not a full router transaction cost.

## Release coordination

Enable `uniswap-v4-evplusai`, onboard the pool, deploy this library revision, and
configure fee/expiry refresh. Confirm production route API discovery and execution
simulation afterward. A merged PR or registry listing alone does not prove that
the production router has activated a pool, or that its price will win a route.
