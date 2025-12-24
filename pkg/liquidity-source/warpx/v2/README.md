# WarpX Integration

This package implements the liquidity source integration for **WarpX** (a Uniswap V2 fork on MegaETH).

## Configuration

To enable this source, add the following configuration to your `config.json` (or equivalent):

```json
"warpx": {
  "dexID": "warpx",
  "factoryAddress": "0xB3Ae00A68F09E8b8a003B7669e2E84544cC4a385",
  "routerAddress": "0x2Fcb75d7dDea4CEa87e77Aa69Fb9d3c4CD93Deb5",
  "fee": 30,
  "feePrecision": 10000,
  "newPoolLimit": 100
}
```

## Protocol Details
- **Chain**: MegaETH
- **Type**: Uniswap V2 Fork
- **Fee**: 0.3% (30/10000)
- **Support**: Standard CPMM (x * y = k)

## Maintainers
- WarpX Module Team
