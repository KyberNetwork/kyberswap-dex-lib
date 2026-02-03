# Stabull DEX Integration for KyberSwap

This directory contains the integration of Stabull DEX as a liquidity source for the KyberSwap aggregator.

## 🏗️ Integration Status: TEMPLATE/SKELETON

**⚠️ IMPORTANT:** This is a skeleton implementation that requires customization with your actual Stabull contract details.

## 📁 Files Created

- **constant.go** - Constants, method names, event signatures, gas costs
- **config.go** - Configuration structure for factory addresses and settings
- **type.go** - Type definitions for pool state, reserves, metadata
- **pools_list_updater.go** - Discovers new pools from PoolFactory contracts
- **pool_tracker.go** - Tracks pool state changes via events and RPC calls
- **pool_simulator.go** - Simulates swap outputs for routing optimization
- **abis.go** - ABI parsing and initialization
- **embed.go** - Embeds ABI JSON files
- **abis/*.json** - Contract ABI files (placeholders)

## ✅ What's Implemented

1. ✅ Basic structure following kyberswap-dex-lib patterns
2. ✅ Pool discovery from factory contract
3. ✅ Pool state tracking framework
4. ✅ Event monitoring structure (Swap, Deposit, Withdraw, Oracle updates)
5. ✅ Swap simulation framework
6. ✅ ABI loading mechanism

## ⚙️ What Needs Customization

### 1. Contract ABIs (CRITICAL)

Replace the placeholder ABIs in `abis/` with your actual contract ABIs:

- **StabullFactory.json** - Your PoolFactory contract ABI
- **StabullPool.json** - Your Pool contract ABI  
- **ChainlinkAggregator.json** - Update if using custom oracle

**How to get ABIs:**
```bash
# Option 1: From Etherscan/Block Explorer
# Go to Contract -> Code -> Contract ABI

# Option 2: From compiled artifacts
# Copy from your build/artifacts directory

# Option 3: Using cast (Foundry)
cast interface <contract_address> --chain <chain_name>
```

### 2. Method Names (`constant.go`)

Update these with your actual contract method names:

```go
// Factory methods
factoryMethodAllPoolsLength = "allPoolsLength" // Update this
factoryMethodGetPool        = "allPools"       // Update this

// Pool methods
poolMethodGetReserves = "getReserves" // Update this
poolMethodToken0      = "token0"      // Usually standard
poolMethodToken1      = "token1"      // Usually standard
poolMethodGetRate     = "getRate"     // Update with your oracle rate method
poolMethodGetFee      = "swapFee"     // Update this
```

### 3. Event Signatures (`constant.go`)

Update with your actual event signatures:

```go
eventSwap     = "Swap(address,uint256,uint256,uint256,uint256,address)"
eventDeposit  = "Deposit(address,uint256,uint256)"
eventWithdraw = "Withdraw(address,uint256,uint256)"
```

Get event signatures from your contracts or using:
```bash
cast sig-event "Swap(address,uint256,uint256,uint256,uint256,address)"
```

### 4. Pricing Logic (`pool_simulator.go`)

**CRITICAL:** Replace the placeholder swap calculation in `calculateSwap()` with your actual pricing formula.

Current placeholder uses simple constant product (x*y=k). You need to implement:
- Your actual swap calculation formula
- How oracle rates affect pricing
- Fee calculations
- Any slippage or price impact logic

Example questions to answer:
- Is it constant product, stable swap, or custom?
- How does the Chainlink oracle rate factor in?
- Are fees percentage-based or fixed?
- Any min/max swap amounts?

### 5. Pool State Structure (`type.go`)

Update `Extra` struct with all state variables needed for swap simulation:

```go
type Extra struct {
    OracleRate *big.Int `json:"oracleRate"`
    SwapFee    *big.Int `json:"swapFee"`
    
    // Add your additional state variables:
    // LastUpdateTime uint64 `json:"lastUpdateTime"`
    // ProtocolFee *big.Int `json:"protocolFee"`
    // etc.
}
```

### 6. Event Decoding (`pool_tracker.go`)

Implement these event decoding functions:

- `findLatestSwapEvent()` - Parse Swap events from logs
- `decodeSwapEvent()` - Extract new reserves from Swap event
- `findLatestDepositEvent()` - Parse Deposit events
- `decodeDepositEvent()` - Extract reserves from Deposit
- `findLatestWithdrawEvent()` - Parse Withdraw events  
- `decodeWithdrawEvent()` - Extract reserves from Withdraw
- `findLatestOracleUpdate()` - Parse Chainlink AnswerUpdated events
- `decodeOracleUpdate()` - Extract new rate from oracle event

### 7. Gas Costs (`constant.go`)

Measure and update actual gas costs:

```go
defaultGas = Gas{
    Swap: 150000, // Update with actual measured gas
}
```

Measure by:
1. Performing test swaps on testnet
2. Checking gas used in transaction receipt
3. Add 10-20% buffer for safety

### 8. Reserves Structure (`type.go`)

If your pool tracks more than just Reserve0/Reserve1:

```go
type Reserves struct {
    Reserve0 *big.Int
    Reserve1 *big.Int
    // Add any additional fields your pool tracks
}
```

## 🚀 Integration Steps

### Step 1: Update ABIs
1. Get your PoolFactory ABI
2. Get your Pool ABI
3. Replace placeholder files in `abis/`

### Step 2: Update Constants
1. Fix method names in `constant.go`
2. Fix event signatures in `constant.go`
3. Measure and set gas costs

### Step 3: Implement Pricing Logic
1. Open `pool_simulator.go`
2. Find `calculateSwap()` function
3. Replace with your actual formula
4. Test with known swap examples

### Step 4: Implement Event Parsing
1. Open `pool_tracker.go`
2. Implement all event finding/decoding functions
3. Test with actual transaction logs

### Step 5: Configuration
1. Create config file for each chain
2. Set factory addresses
3. Set oracle addresses if needed

### Step 6: Testing
1. Test pool discovery
2. Test state tracking
3. Test swap simulation against on-chain results

### Step 7: Register with KyberSwap
1. Submit PR to kyberswap-dex-lib
2. Provide documentation
3. Coordinate with KyberSwap team

## 📋 Testing Checklist

- [ ] Pool discovery finds all pools correctly
- [ ] Pool state updates from events work
- [ ] Pool state fetching from RPC works
- [ ] Swap simulation matches on-chain results (within 0.1%)
- [ ] Oracle rate updates are captured
- [ ] Gas estimates are accurate
- [ ] Edge cases handled (zero amounts, insufficient reserves)
- [ ] Multiple chains supported (if applicable)

## 🔗 Resources

- [KyberSwap DEX Integration Guide](https://docs.kyberswap.com/)
- [Example: Uniswap V2 Integration](../uniswap/)
- [Example: Curve Integration](../curve/)
- [Ethereum ABI Encoding](https://docs.soliditylang.org/en/latest/abi-spec.html)

## 📞 Support

For questions about:
- **This template**: Review the comments marked with `// TODO:`
- **KyberSwap integration**: Contact KyberSwap team
- **Your Stabull contracts**: Refer to your contract documentation

## 🎯 Next Steps

1. Review all files marked with `// TODO:` comments
2. Gather your contract ABIs and addresses
3. Understand your pricing formula
4. Start with `constant.go` and work through each file
5. Test thoroughly before submitting

Good luck with your integration! 🚀
