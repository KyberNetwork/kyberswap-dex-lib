# Address Verification Results

## Issue

RPC calls to fetch pool state are returning empty data (`abi: attempting to unmarshal an empty string while arguments are expected`).

## Addresses Tested

### Polygon NZDS/USDC Pool

**User Provided**: `0xdcb7efACa996fe2985138bF31b647EFcd1D0901a`  
**Stabull API**: `0xdcbefACa996fe2985138bF31b647EFcd1D0901a` (note: no '7')

Both addresses return empty data when calling `viewCurve()`, `liquidity()`, and even `name()`.

### Base BRZ/USDC Pool

**User Provided**: `0x8A908aE045E611307755A91f4D6ECD04Ed31EB1B`  
**Stabull API**: `0xce0abd182d2f5844f2a0cb52cfcc55d4ff4fcba` (completely different!)

### Ethereum NZDS/USDC Pool

**User Provided**: `0xe37D763c7c4cdd9A8f085F7DB70139a0843529F3`  
**Stabull API**: Not found in initial API fetch

## Findings

1. **ABI is correct**: The `viewCurve` method exists in `StabullPool.json` with correct signature
2. **Methods are recognized**: Test shows all methods (`viewCurve`, `liquidity`, etc.) are in the ABI
3. **RPC returns empty**: `RawResponse:[]` indicates the eth_call executed but returned no data

## Possible Causes

1. **Wrong addresses**: The addresses might not have the Stabull pool contract deployed
2. **Wrong Alchemy API key**: The key `IqvzEgP3ce5i1ruu_uNyK` might not be valid
3. **Contract version mismatch**: Addresses might point to different version of Stabull
4. **Typo in user-provided addresses**: Note the '7' discrepancy in Polygon address

## Recommendation

Please verify:
1. The Alchemy API key is valid and has credit
2. The exact pool contract addresses are correct
3. Whether these are v1 or v2 contracts (ABI might differ between versions)

To test manually:
```bash
# Using curl to test RPC directly
curl https://polygon-mainnet.g.alchemy.com/v2/YOUR_API_KEY \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"0xdcbefACa996fe2985138bF31b647EFcd1D0901a","data":"0x8c8c96e8"},"latest"],"id":1}'
```

(Note: `0x8c8c96e8` is the function selector for `name()`)
