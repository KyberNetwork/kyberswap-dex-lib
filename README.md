# kyberswap-dex-lib

## What?

KyberSwap uses this lib for off-chain tracking and computation, specifically

1. Fetch pools;
2. Track latest pool states; and
3. Simulate expected output for a given input swap amount

in order to search for the optimal aggregated swapping route.

## How to Contribute?

* Off-chain: Implements 3 things in pkg/liquidity-source (pkg/source contains legacy code using big.Int):

1. PoolsListUpdater: fetches latest pool list incrementally
2. PoolTracker: tracks latest pool state on new log event or on an interval
3. PoolSimulator: simulates expected output for a given input swap amount
  a. Use uint256.Int/int256.Int, not big.Int, for all simulator math and pool state; new big.Int simulators are not accepted
  b. CloneState should also be implemented

* On-chain: Contribute encoding to https://github.com/KyberNetwork/ks-dex-adapter-lib
