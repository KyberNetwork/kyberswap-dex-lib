# kyberswap-dex-lib — Agent Instructions

## Repository Overview

**kyberswap-dex-lib** is a library used by the KyberSwap backend to integrate with decentralized
exchanges.

### Directory Structure

```text
.
├── pkg/
│   ├── liquidity-source/       # New DEX integrations; prefer this for new work
│   │   └── <dex>/              # constant/config/type, lister, tracker, simulator, ABI, tests
│   ├── source/                 # Legacy integrations and shared pool/list/tracker interfaces
│   │   └── pool/               # IPoolSimulator interfaces and simulator factory registry
│   ├── entity/                 # Persisted wire models such as entity.Pool
│   ├── pooltypes/              # Aggregates DEX imports so init() registration runs
│   ├── msgpack/                # Simulator serialization registration and generated bindings
│   ├── swaplimit/              # Shared-inventory limit tracking
│   ├── util/                   # Math, ABI, test helper packages
│   │── valueobject/            # Chain, exchange, address, and token constants
├── go.mod / go.sum             # Go module definition
└── AGENTS.md                   # Agent instructions for this repository
```

Typical new integration shape:

```text
pkg/liquidity-source/<dex>/
├── constant.go                 # DexType, gas constants, sentinel errors
├── type.go                     # Extra, StaticExtra, PoolMeta, SwapInfo, state structs
├── embed.go                    # go:embed wiring for ABI JSON
├── abi.go                      # Parsed ABI handles, call/event helpers if needed
├── abi/                        # Minimal ABI JSON used by the package
├── pool_list_updater.go        # Pool discovery and metadata cursor handling
├── pool_tracker.go             # Full/event-driven state refresh into entity.Pool
├── pool_simulator.go           # Pure in-memory pricing and UpdateBalance
├── math.go                     # On-chain math port when complex enough to isolate
└── *_test.go                   # Math, simulator, tracker, lister, and verification tests
```

## DEX Integration & Best Practices

### Protocol Exploration

- Confirm contracts are real and callable.
- Verify source facts against deployed behavior. Explorer source is preferred, docs and repos are hints.
- Identify the pool discovery mechanism.
- Identify the exact swap calculation logic, including pricing, fees, rounding, operation order, etc.
- Identify shared inventory/vault behavior early.
- Check native token support.
- Decide whether to reuse an existing integration or create a new one.

### Pool Discovery - IPoolsListUpdater

- Prefer discovering pools on-chain over off-chain indexes.
- Do not fetch or set reserves during discovery. Use 0 as a placeholder; the tracker will update the real reserves later.
- Do not fetch token symbol/decimals; they will be populated after pool listing. Assume decimals are available for the pool tracker/simulator. Only set the token address and swappable: true.
- Lowercase all pool addresses and token addresses.
- Always wrap native tokens when the protocol supports native swaps (some protocols use the zero address or 0xeeee...eeee).
- Persist immutable metadata the tracker and simulator need in `StaticExtra` of `entity.Pool`

### Pool Tracker - IPoolTracker

- Track reserves and keep reserve order aligned with token order.
- Batch independent reads via multicall when the chain and ABI support it.
- Pin all multicalls to the same block when a refresh needs multiple multicalls to keep pool state consistent.
- Store mutable state in `Extra`, immutable state in `StaticExtra`.
- Track all pool state required by the simulator for `CalcAmountOut` and `UpdateBalance`, including every flag/check used by the on-chain swap path (e.g. paused, swapEnabled, caps/limits, etc.) so the simulator can reject swaps that would revert on-chain.

### Pool Simulator - IPoolSimulator

- `CalcAmountOut` must be pure and must not mutate state on success or failure.
- `UpdateBalance` must consume the `SwapInfo` returned by `CalcAmountOut`; never recompute swap results.
- `CloneState()` must deep-copy every slice/map/pointer `UpdateBalance` writes by index or in place; shallow already covers value arrays and fields reassigned wholesale (copy-on-write).
- Repeated quoting and repeated state updates must behave deterministically.
- Mirror on-chain integer math exactly to the wei.
- Always perform overflow checks and ensure state and swap results do not share mutable integer pointers.
- Successful quotes must return positive output amounts; zero-output swap should return an error.
- If a swap does not consume the full input amount, the remaining amount must be returned.
- Gas estimation should reflect the actual swap execution cost.
- Prefer `uint256.Int`/`int256.Int` over `big.Int` for math and state, and reuse helpers in `pkg/util` (`big256`, `int256`, ...) instead of reimplementing.
- Test integration correctness, including pricing, limits, and protocol constraints. Port all relevant on-chain or documented math test cases.

**Optional capabilities**: 
- Implement `CalculateLimit()` and support `pool.SwapLimit` when the protocol has shared inventory/vault behavior.
- `IPoolExactOutSimulator`: Implement `CalcAmountIn` only when the protocol supports exact-out swaps.
- `IPoolSupportNativeSwap`: Implement `SupportsNativeSwap()` when the protocol supports native token.

### Registration & wiring

Register the simulator, lister, and tracker factories (keyed by `DexType`; double registration panics). Also required:

1. Add an exchange constant in `pkg/valueobject`.
2. Add the dex type to `pkg/pooltypes`.
3. Run `go generate ./pkg/msgpack/...` to register the simulator type for serialization.

## uint256 Performance Rules

### Stack allocation
- `var x uint256.Int` stays on stack. `new(uint256.Int)` always heap-allocates.
- Return by value (`(hi, lo uint256.Int)`) avoids heap escape for multi-return.
- `&localVar` passed to a function stays on stack as long as the function doesn't store the pointer.

### Aliasing safety
These ops read all inputs before writing the result, so z==x or z==y is safe:
`Add`, `Sub`, `Mul`, `Div`, `Lsh`, `Rsh`, `And`, `AddOverflow`, `MulDivOverflow`.

### Never use MulMod to compute the high word of a 512-bit product
- `MulMod(x, y, UMax)` triggers `Reciprocal(UMax)` (Barrett reduction precompute) + `reduce4` (5×5 multiply) inside holiman/uint256 — expensive CPU.
- Use 128-bit split: split x,y into 128-bit halves, compute cross-terms, track 256-bit overflow carries with `AddOverflow`, add each carry as `u256.U2Pow128` to hi. See `pkg/liquidity-source/carbon/match.go:mul512` for the full implementation.
- `AddOverflow(x, y)` returns `(*Int, bool)`; ignore the `*Int` with `_` when you just need the carry bool.

### MulDivUp/Down are zero-alloc
- `big256.MulDivUp/Down` use a stack-allocated `[8]uint64` internally — no heap. Safe to call in hot loops.
- Prefer over `MulMod`-based remainder checks.

## Pull Request

Create a PR from your feature branch to **kyberswap-dex-lib** `main`.
The PR must contain brief explanation about the DEX background, pricing logic, links to existing documentation, important contract addresses, and anything you think could help us review your code faster.
Add supporting evidence when available, such as explorer links, sample transactions, contract sources, fixtures, or quote comparison results.
