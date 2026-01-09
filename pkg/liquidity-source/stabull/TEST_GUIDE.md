# Stabull Integration Test Guide

This guide explains how to run the test harnesses to validate the Stabull implementation against actual on-chain behavior.

## Prerequisites

1. RPC endpoints for the chains you want to test (Polygon, Base, Ethereum)
2. Known Stabull pool addresses (get from factory events or Stabull documentation)
3. Go test environment setup

## Test Structure

### 1. Pool List Updater Tests (`pools_list_updater_test.go`)

Validates that pool discovery is working correctly.

**Note**: These tests will likely show that the current implementation needs event-based discovery, as the factory doesn't support indexed enumeration.

**Run:**
```bash
# Run with actual RPC calls
go test -v -run TestPoolsListUpdater

# Skip long-running tests
go test -v -short -run TestPoolsListUpdater
```

**Expected Output:**
- Warning that factory uses event-based discovery
- Pool details if any are discovered via alternate methods

### 2. Pool Tracker Tests (`pool_tracker_test.go`)

Validates that pool state fetching is correct (reserves + greeks + oracle rates).

**Before running**: Update test cases with actual pool addresses:
```go
poolAddress: "0xYOUR_ACTUAL_POOL_ADDRESS",
```

**Run:**
```bash
# Test state fetching
go test -v -run TestPoolTracker_FetchPoolStateFromNode

# Test full pool state update
go test -v -run TestPoolTracker_GetNewPoolState
```

**Expected Output:**
```
=== Pool State ===
Pool: 0x8a9008ae045048cf6e1821443cf1be4a411a0994
Reserves:
  Reserve 0: 1234567890000000000000
  Reserve 1: 9876543210

Curve Parameters (Greeks):
  Alpha (α): 500000000000000000
  Beta (β): 350000000000000000
  Delta (δ): 500000000000000000
  Epsilon (ε): 1500000000000000
  Lambda (λ): 1000000000000000000

Oracle Information:
  Base Oracle: 0xOracle1Address
  Base Rate: 1500000000000000000
  Quote Oracle: 0xOracle2Address
  Quote Rate: 1000000000000000000
  Derived Oracle Rate: 1500000000000000000
```

**Validation Checklist:**
- ✅ All reserves are positive numbers
- ✅ All curve parameters (α, β, δ, ε, λ) are positive
- ✅ Parameters are in expected ranges (typically 1e15 - 1e18 scale)
- ✅ Oracle addresses are valid (if configured)
- ✅ Oracle rates are reasonable (typically 1e18 scale)

### 3. Pool Simulator Tests (`pool_simulator_test.go`)

**Most Critical Test** - Validates that swap calculations match the contract's `viewOriginSwap` output.

**Before running**: Update test cases with actual data:
```go
{
    name:            "Polygon NZDS/USDC - Small swap",
    rpcURL:          "https://your-polygon-rpc.com",
    poolAddress:     "0xYOUR_POOL_ADDRESS",
    tokenIn:         "0xNZDS_TOKEN_ADDRESS",
    tokenOut:        "0xUSDC_TOKEN_ADDRESS",
    amountIn:        "1000000000000000000", // 1 token
    maxDeviationBps: 100, // 1% tolerance
},
```

**Run:**
```bash
# Run the critical validation test
go test -v -run TestPoolSimulator_CalcAmountOut_ValidateAgainstContract

# Run all simulator tests
go test -v -run TestPoolSimulator
```

**Expected Output:**
```
=== Fetching pool state from chain ===
Pool State:
  Reserve 0: 1000000000000000000000
  Reserve 1: 1000000000
  Alpha: 500000000000000000
  ...

=== Calling contract viewOriginSwap ===
Contract viewOriginSwap:
  Input: 1000000000000000000
  Output: 998500000

=== Calculating with pool simulator ===
Simulator CalcAmountOut:
  Input: 1000000000000000000
  Output: 997800000

=== Comparison ===
Contract Output:  998500000
Simulator Output: 997800000
Difference:       700000
Deviation:        70 bps (0.70%)
Max Allowed:      100 bps (1.00%)
✅ PASS - Deviation within acceptable range
```

**Success Criteria:**
- ✅ Deviation < maxDeviationBps for all test cases
- ✅ Simulator output is positive
- ✅ Simulator output < available reserves
- ✅ Larger swaps have proportionally higher slippage

**If tests fail:**
1. Check if oracle rates are being used (log should show oracle rate)
2. Verify curve parameters are correct
3. Check if the math implementation needs adjustment
4. Try different deviation tolerances for different swap sizes

## How to Find Pool Addresses

### Method 1: From Factory Events
```bash
# Using cast (foundry)
cast logs --rpc-url <RPC_URL> \
  --from-block <START_BLOCK> \
  --address <FACTORY_ADDRESS> \
  --topic0 0xe7a19de9e8788cc07c144818f2945144acd6234f790b541aa1010371c8b2a73b
```

### Method 2: From Stabull Documentation
Check the Stabull docs or DefiLlama adapter for known pools.

### Method 3: From Block Explorer
Look at the factory contract's transaction history for NewCurve events.

## Debugging Failed Tests

### If `pool_tracker` tests fail:
1. Check RPC endpoint is working: `curl <RPC_URL> -X POST -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'`
2. Verify pool address exists on that chain
3. Check ABI is correct (compare with verified contract)

### If `pool_simulator` tests show high deviation:
1. **Check oracle integration**: Oracle rates should be logged
2. **Verify greek parameters**: Compare with contract state
3. **Test with smaller amounts**: Large swaps may have different curve behavior
4. **Check token decimals**: Ensure correct decimals for both tokens
5. **Review math.go**: The approximation may need refinement

## Iterating on the Implementation

Recommended workflow:
1. Run `TestPoolTracker` first - ensure state fetching works
2. Run `TestPoolSimulator_ValidateAgainstContract` with small amounts
3. Adjust `math.go` based on deviation results
4. Gradually test larger swap amounts
5. Repeat until deviation is consistently < 1-2%

## Test Configuration

Update these in the test files:

### RPC URLs
```go
rpcURL: "https://your-rpc-endpoint.com"
```

### Pool Addresses
Get from:
- Stabull documentation
- Factory NewCurve events
- Block explorer

### Deviation Tolerances
- Small swaps (<1% of liquidity): 50-100 bps (0.5-1%)
- Medium swaps (1-10% of liquidity): 100-200 bps (1-2%)
- Large swaps (>10% of liquidity): 200-500 bps (2-5%)

## Example Test Run

```bash
# Full test suite with actual RPC calls
go test -v -timeout 5m ./pkg/liquidity-source/stabull/...

# Just the critical validation test
go test -v -run TestPoolSimulator_CalcAmountOut_ValidateAgainstContract

# Quick unit tests only
go test -v -short ./pkg/liquidity-source/stabull/...
```

## Success Metrics

✅ **Ready for production when:**
- Pool tracker fetches all greeks correctly
- Oracle rates are properly integrated
- Simulator deviation < 1% for typical swap sizes
- Simulator deviation < 5% for large swaps
- All validation tests pass consistently

## Notes

- Oracle rate integration is critical - lambda parameter weights oracle influence
- The math is an approximation - some deviation is expected
- Stabull uses complex curve math - perfect accuracy may not be achievable
- Focus on getting "close enough" for routing decisions (< 2% deviation)
