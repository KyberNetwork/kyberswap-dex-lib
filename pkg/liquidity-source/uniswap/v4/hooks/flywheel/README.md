# Flywheel native V4 hook

The October 13, 2026 deployment uses ordinary Uniswap V4 concentrated liquidity
from the first trade. This hook adapter adds its immutable fee to Kyber's existing
V4 simulator. It does not use the older `flywheel-fun` settlement contract,
synthetic bonding-curve reserves, or internal ETH routing. The older source stays
available for launches from previous deployments.

## Deployment and discovery

Robinhood Chain, chain ID **4663**, deployment starts at block **87735162**.

| Contract | Address |
| --- | --- |
| Factory | `0xf05a26e92e27b8f0385f6895f5507db26a2f4b8d` |
| Hook | `0x992118de23b4ebe9f527dacbbb7d8d4bbbf8e044` |
| PoolManager | `0x8366a39cc670b4001a1121b8f6a443a643e40951` |
| StateView | `0xf3334192d15450cdd385c8b70e03f9a6bd9e673b` |
| V4Quoter | `0x8dc178efb8111bb0973dd9d722ebeff267c98f94` |
| UniversalRouter | `0x8876789976decbfcbbbe364623c63652db8c0904` |
| Permit2 | `0x000000000022d473030f116ddee9f6b43ac78ba3` |

Use the normal V4 pool discovery source, including PoolManager `Initialize`
events with the hook above. Enable exchange `uniswap-v4-flywheel` in Robinhood's
pool/router-service configuration. Each pool ID is the hash of its PoolKey;
neither the factory nor the singleton PoolManager address is a pool ID.
`TokenLaunched` and `hook.poolKey(token)` provide an independent discovery check.

PoolKeys have sorted ERC20 currencies, LP fee **0**, tick spacing **200**, and
empty hook data. Native ETH is WETH in the pool. Arbitrary supported ERC20 pair
currencies do not require separate hook registrations or hardcoded quote tokens.
The standard router chooses and executes any external hops.

Only registered factory pools can initialize this hook. Its fee is fixed for all
of them, so `Track` needs no additional contract calls. Reuse the normal V4
tracker's slot0, active liquidity, complete initialized ticks and pool-specific
reserves. Do not use token balances of the shared PoolManager as pool reserves.
An untouched one-sided launch can have zero active liquidity at its upper tick;
its initialized ticks still allow the first buy.

## Pricing and execution

For exact-input execution, the core V4 swap runs normally and the hook subtracts
`floor(grossOutput * 400 / 10000)` from the output. Both directions have this fee.
Fee claims are recorded separately; reward conversion does not happen inside the
user swap. External LP positions are permitted and must be included by the V4
tracker.

Kyber's `CalcAmountIn` reverses **exact-input execution**. It therefore grosses
up the desired net output using `floor((net - 1) * 25 / 24) + 1` before applying
the normal core inverse quote. It must not add 4% to the input. The deployed
contract's actual exact-output operation instead charges its unspecified input
leg; that is a different operation and is not what this reverse quote represents.
The simulator-only `BeforeSwap` adjustment does not imply an on-chain
`beforeSwap` callback: deployed permissions are `0x2044`.

Graduation marks a threshold reached by the locked position's quote principal.
It does not replace the pool, change pricing, or change the hook fee. The native
V4 tick engine handles trading both before and after it. Maintenance can add
liquidity later; normal `ModifyLiquidity` tracking remains necessary.

Execution uses the existing V4 executor path and empty hook data. Robinhood's
deployed UniversalRouter includes `minHopPriceX36` fields in its command tuples;
the included fork test uses that deployed ABI. This is a chain-router ABI detail,
not a Flywheel-specific swap interface. A router that calls PoolManager directly
uses the standard V4 swap and settlement interfaces.

## Verification and activation

The recorded fork checks deployment runtime hashes, then compares ten quotes to
actual local UniversalRouter executions: WETH/USDG/CASHCAT buys and sells, first
buy, single-buy graduation, and post-graduation buys/sells. Expected outputs and
subsequent state/quotes match the simulator to the wei. Tests also cover minimum
input inversion, rounding boundaries, liquidity-exhaustion rejection, failed-quote purity, clone
isolation, wrong-chain rejection, and production msgpack round trips.

Ordinary Go tests are offline. Fixture generation uses a read-only public RPC
proxy with a bounded cache and executes transactions only on a local fork. No
mainnet transactions or paid Alchemy calls are needed by the test suite. The
130,000 additional gas estimate has headroom over the recorded maximum 89,306
gas inside `afterSwap`; it is not a transaction gas limit.

Kyber still needs to enable discovery/routing for this exchange and validate its
own production executor. Passing the canonical-router fork tests does not claim
that Kyber's production route service is already enabled.

Sources: [deployment, verified sources and ABI package](https://flywheel.cash/integrations/20261013/index.html),
[trading rules](https://flywheel.cash/integrations/20261013/TRADING.md),
[migration checklist](https://flywheel.cash/integrations/20261013/codex-migration.html),
[factory explorer](https://robin.etherscan.io/address/0xf05a26e92e27b8f0385f6895f5507db26a2f4b8d#code),
[hook explorer](https://robin.etherscan.io/address/0x992118de23b4ebe9f527dacbbb7d8d4bbbf8e044#code).
