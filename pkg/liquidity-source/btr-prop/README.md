# btr-prop

Quotes BTR's oracle-priced AIMM pool by sampling the pool's `getSwapQuote` view into a per-pair ladder.

- The tracker scales every sampled output down by the worst-case growth of the pool's staleness premium over `MaxAge + InclusionAllowance`. The premium constants in `risk.go` (`staleZ`, `maxStaleAllowance`, the sqrt law, the half-spread charge) mirror pool parameters: re-check them against the pool whenever it is upgraded.
- Checks that only `swap_qe` runs (feed gate, depeg band, route feed rule, liquid floor, swap gate) are mirrored from state read in the same multicall; a pair that would revert is refused.
- `GasPerExtraHop` is an unmeasured estimate.
