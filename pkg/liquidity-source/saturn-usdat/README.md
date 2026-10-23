# saturn-usdat

Saturn's USDat on Ethereum mints 1:1 against USDC through the SwapFacility. One static pool,
address = USDat, tokens `[USDC, USDat]`; only USDC -> USDat exists on-chain (the reverse reverts).

| Contract | Address |
|---|---|
| SwapFacility (proxy, swap router and USDC spender) | `0x0bc305e7e13113caed3f5486849e9518a1cc4173` |
| USDat (MultiMint extension, proxy) | `0x23238f20b894f29041f48d88ee91131c395aaa71` |
| USDat implementation | [`0x496a4a33b6181f4536203488d9a05ac1429e702c`](https://etherscan.io/address/0x496a4a33b6181f4536203488d9a05ac1429e702c#code) |

## Pricing

`SwapFacility.swap(USDC, USDat, amount, recipient)` wraps USDC with no fee. The simulator ports
`MultiMint._convertAmounts` (floor when scaling down, checked multiply when scaling up) and the
cap check `assetCap >= assetBalanceOf + amount`. Decimals dust is refunded and returned as
`RemainingTokenAmountIn`. `UpdateBalance` adds the pulled USDC to `assetBalanceOf`.

The tracker reads the cap, balance, decimals, allowed-asset, pause and approved-extension flags in
one multicall. Reserves report the remaining capacity on both sides, or zero when a flag would
revert the swap.

## Whitelist

USDat has a per-account whitelist (`isWhitelistEnabled`) on both the caller and the recipient of
the mint. On a single hop both are the executor that calls the facility, which then forwards USDat
to the user by plain transfer (not whitelist-gated). Set `executor` in the config to have the
tracker read `isWhitelisted(executor)`: while the whitelist is on and the executor is not on it,
the simulator rejects quotes and reserves drop to zero instead of quoting swaps that revert with
`AccountNotWhitelisted`. Without `executor`, the whitelist is not checked.
