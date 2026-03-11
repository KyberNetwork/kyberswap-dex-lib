# lunarbase

`lunarbase` is implemented as a standard `kyberswap-dex-lib` liquidity source.

Implemented components:

1. `PoolsListUpdater`
2. `PoolTracker`
3. `PoolSimulator`

Notes:

- The source is wired to the current Base deployment defaults:
  - core: `0xeccd5b11549140c67fa2e8b6028bc86a4f9cab6d`
  - periphery: `0x110ab7d4a269cc0e94b6f56926186ec4716edb1b`
  - permit2: `0x000000000022d473030f116ddee9f6b43ac78ba3`
- `X()` resolves to the native token on the live pool, so the entity pool stores wrapped native for routing compatibility.
- `PoolSimulator` uses exact core quote calls with `stateOverride` over storage slot `0x2` (`latestUpdateBlock | fee | pX96`) and slot `0x3` (`concentrationK | reserveY | reserveX`).
- Router execution is still performed via the LunarBase periphery. For ERC20 inputs the approval target is Permit2, not the periphery.
