# flywheel-fun

Flywheel launchpad on Robinhood chain (4663), replacement factory `0xe7743b4039dbcd05c5242939aa8db274c65fcbfa`, discovery start block **82181683**.
Contracts, ABIs, sources and mainnet receipts: https://flywheel.cash/integrations/20261007/index.html

## Pricing

Every trade goes through `NativeTradeSettlement` `0xad06b86264411e0278dbcebce556c913b44da004` (`buy`/`buyWithRefund`/`sell`, ETH in or out). Direct factory/V4 swaps revert at the hook. One pool = one launch, WETH <-> launch token. The settlement:

1. Optionally swaps ETH through one configured external Uniswap pool and up to two graduated Flywheel parents (the route).
2. Trades the bonding curve, or after graduation the hook-locked canonical V4 pool. The native hook charges no LP fee; directional Uniswap protocol fees are offset against the platform allocation.
3. Takes NativeFeeMath fees, ported exactly (512-bit mul/div, Solidity rounding).

A buy that crosses graduation fills the curve up to the threshold, reverses the unused quote through the route and refunds the ETH as `RemainingTokenAmountIn`. The new canonical pool needs a refresh before the next quote.

The replacement curve getter appends `virtualTokenOffset` and `curveInvariant`. Use its fixed invariant rather than multiplying rounded current reserves. Virtual tokens are pricing units; real inventory is `tokenReserve - virtualTokenOffset`. At graduation the tracked unsold tokens and real quote backing enter the locked V4 position, with bounded rounding remainder donated to that pool.

Every Flywheel market crossed charges its own fees. Parent trades and refund sales also charge their market's allocation. The serialized `SwapInfo.route` is already the settlement route; pass it unchanged as the `route` (and, for a refund, `refundRoute`) argument of the settlement. For native parents it is `abi.encode(bytes4("FWL1"), address[] parentsNearestFirst, bytes externalRoute)`. Use `buyWithRefund` only when `SwapInfo.Refunds` is set (it reverts without a refund) and plain `buy` otherwise (it reverts on a partial fill). Slippage and deadline belong in the executor's calldata builder.

## Route base pools

`quoteBasePools` maps a pairing token to the uniswapv3 pool address or uniswap-v4 pool id the settlement swaps through. WETH-paired launches need none. The lister copies the id into `StaticExtra.basePools`; router-service loads that indexed pool and passes it in via BasePoolMap, so the route adds no duplicate source. The simulator quotes the hop with the base's own simulator, deep-copies it on clone (like curve meta pools) and rebuilds the settlement route bytes from it: a V3 address, or a hook-free V4 key that must hash to the pool id. A missing base fails with `ErrBasePool`.

```json
"quoteBasePools": {
  "0x73c2de14c7fa0a57cc2d9722b959ea70b881ffe4": "0x8742f10cc122395bfb79eb2dc51fb15b97b297759a5f79b5f2463d38af59083d",
  "0x39dbed3a2bd333467115de45665cc57f813c4571": "0xed50bdeea8adc232f159486192a4157281d722ff"
}
```

The tracker reads curve and authenticated canonical-market state at one block, discovers the parent chain from this factory, and reads the external base's spot price to value quote reserves in WETH. Configure `quoteBasePools` for the external leaf, including when it is behind native parents. External bases are indexed and refreshed independently, so their block may differ from the Flywheel pool's; the settlement re-prices them at execution.

Native parent pools and a graduated token's own canonical pool are exposed via `GetBasePools` and can be relinked via `SetBasePool`. Router-service only relinks the indexed external bases (`basePools`), so each pool keeps its own copy of parent and canonical liquidity. Clones own all mutable liquidity; stale quote replay is validated entirely before any shared pool is modified. Same-factory graduated parents only, maximum depth two, no cycles.

## Tests

- 390 fee vectors from the frozen Solidity fee library; curve, simulator, tracker and lister tests; quote purity fuzzing (`-fuzz '^FuzzCompositeQuotePurity$'`).
- Real replacement snapshots cover indexed V3/V4 bases, one/two native parents, shared liquidity, clone isolation and msgpack serialization.
- `TestLocalForkQuoteExecutionParity` is opt-in: `testdata/runner/run-forks.cjs` (set `FLYWHEEL_RPC_URL`; an archive RPC and `FLYWHEEL_FORK_BLOCK` pin the fork) starts a local Anvil fork behind a read-only proxy and calls the settlement directly, as executeGeneric does (`buy`, `buyWithRefund` when `Refunds`, `sell`). Buy/sell outputs and refunds must match executed balance deltas exactly, including graduation refunds and one/two-parent routes. A test-only external-pool provider stands in for Kyber's pool service; production uses indexed base pools.
