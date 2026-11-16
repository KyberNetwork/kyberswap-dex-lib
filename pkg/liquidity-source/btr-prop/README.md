# btr-prop

Quotes BTR's oracle-priced AIMM pool by sampling the pool's `getSwapQuote` view into a per-pair ladder.

- The pool's quote drifts with block time: each non-base leg's staleness premium grows with the square root of its mark's age until the next keeper push. The tracker stores the ladder as sampled, with the sample's block time, each leg's premium inputs (mark time, sigma, allowance) and each pair's legs and sampled spread. At quote time the simulator scales the output by the premium's worst-case growth (no push in between) from the sample to the quote's actual age plus `InclusionAllowance`, the measured time a routed swap takes to land.
- `MaxAge` is a drop rule only: a ladder older than it at quote time is not quoted.
- The premium constants in `risk.go` (`staleZ`, `maxStaleAllowance`, the sqrt law, the half-spread charge) mirror pool parameters: re-check them against the pool whenever it is upgraded.
- Checks that only `swap_qe` runs (feed gate, depeg band, route feed rule, liquid floor, swap gate) are mirrored from state read in the same multicall; a pair that would revert by the end of `MaxAge + InclusionAllowance` is refused.
- `GasPerExtraHop` is an unmeasured estimate.
