# btr-prop

Quotes BTR's oracle-priced AIMM pool by sampling the pool's `getSwapQuote` view into a per-pair ladder.

- The pool's quote drifts with block time: each non-base leg's staleness premium grows with the square root of its mark's age until the next keeper push. The tracker stores the ladder as sampled, with the sample's block time, each leg's premium inputs (mark time, sigma, allowance) and each pair's legs and sampled spread. At quote time the simulator scales the output by the premium's worst-case growth (no push in between) from the sample to the quote's actual age plus `InclusionAllowance`, the measured time a routed swap takes to land.
- `MaxAge` is a drop rule only: a ladder older than it at quote time is not quoted.
- The premium constants in `risk.go` (`staleZ`, `maxStaleAllowance`, the sqrt law, the half-spread charge) mirror pool parameters: re-check them against the pool whenever it is upgraded.
- Checks that only `swap_qe` runs (feed gate, depeg band, route feed rule, liquid floor, swap gate) are mirrored from state read in the same multicall; a pair that would revert by the end of `MaxAge + InclusionAllowance` is refused.
- Depth is liquid cash plus what the pool recalls from the leg's yield hook inside the swap (`PoolHooksLib.preOutflow`, hook slot flag `HOOK_PRE_OUTFLOW`; the pool already prices on full reserves): room = liquid + min(`getInvested`, hook `venueStatus().withdrawableNow`) - `minLiquidity`. It sizes the sample grid and cuts the ladder; `getSwapQuote` stays the price truth, so a size the pool cannot honour reverts and gets no sample. No hook, no flag, an unreadable or distressed hook: liquid only.
- Recall gas (`RecallGas*`, measured on a Monad fork) is added to a quote only when its output, with what earlier quotes took, passes the leg's liquid-only room: Aave 430k, ERC4626/Euler 320k (Morpho taken as ERC4626), Pendle 860k, any other hook kind the max. The kind is read from the hook's own getters (`aavePool`, `vault`, `morpho`, `market`).
- Pendle legs stay liquid only: its recall reverts on an exit above 5 bps of the market and the quote probe prices on full reserves, so it cannot show which recall-backed sizes execute.
- `GasPerExtraHop` is an unmeasured estimate.
