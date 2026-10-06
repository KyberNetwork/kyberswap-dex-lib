# flywheel-fun

Flywheel launchpad on Robinhood chain (4663), native-fee factory 0xee54da52128dd851c71b1c58d371966231b66c40.
Contracts, ABIs and mechanics: https://flywheel.cash/integrations/20260930/index.html

## Pricing

Every trade goes through `NativeTradeSettlement` 0x04111c295399582b2b702ad5de8d11be2b50dd5d (`buyWithRefund`/`sell`, ETH in or out). Direct factory/V4 swaps revert at the hook. One pool = one launch, WETH <-> launch token. The settlement:

1. Optionally swaps ETH <-> the launch's pairing token through one configured Uniswap pool (the route).
2. Trades the bonding curve, or after graduation the hook-locked canonical V4 pool. The native hook charges no LP fee; directional Uniswap protocol fees are offset against the platform allocation.
3. Takes NativeFeeMath fees, ported exactly (512-bit mul/div, Solidity rounding).

A buy that crosses graduation fills the curve up to the threshold, reverses the unused quote through the route and refunds the ETH as `RemainingTokenAmountIn`. The new canonical pool needs a refresh before the next quote.

## Route base pools

`quoteBasePools` maps a pairing token to the uniswapv3 pool address or uniswap-v4 pool id the settlement swaps through. WETH-paired launches need none. The lister copies the id into `StaticExtra.basePools`; router-service loads that indexed pool and passes it in via BasePoolMap, so the route adds no duplicate source. The simulator quotes the hop with the base's own simulator, deep-copies it on clone (like curve meta pools) and rebuilds the settlement route bytes from it: a V3 address, or a hook-free V4 key that must hash to the pool id. A missing base fails with `ErrBasePool`.

```json
"quoteBasePools": {
  "0x73c2de14c7fa0a57cc2d9722b959ea70b881ffe4": "0x8742f10cc122395bfb79eb2dc51fb15b97b297759a5f79b5f2463d38af59083d",
  "0x39dbed3a2bd333467115de45665cc57f813c4571": "0xed50bdeea8adc232f159486192a4157281d722ff"
}
```

The tracker reads curve and canonical-market state, plus the base's spot price to value the quote reserve in WETH.

## Tests

- 390 fee vectors from the frozen Solidity fee library; curve, simulator, tracker and lister tests; quote purity fuzzing (`-fuzz '^FuzzCompositeQuotePurity$'`).
- `TestLocalForkQuoteExecutionParity` is opt-in: `testdata/runner/run-forks.cjs` starts a local Anvil fork behind a read-only proxy (set `FLYWHEEL_RPC_URL`). It covers WETH-paired launches; routed launches need the indexed base pools and are verified end to end through router-service.
