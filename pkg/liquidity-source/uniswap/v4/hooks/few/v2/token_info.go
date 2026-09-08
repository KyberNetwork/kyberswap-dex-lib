package few_v2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// fewTokens lists Ring Protocol's currently-published FewToken wrapper pools on
// Ethereum mainnet. Each wrapper pool is a Uniswap v4 pool with fee=0,
// tickSpacing=1, pairing an underlying asset 1:1 against its FewToken wrapped
// representation via a dedicated hook (FewTokenHook / FewUSDTHook / FewETHHook).
//
// Hook source (FewTokenHook, FewUSDTHook, FewETHHook) confirms the wrap/unwrap
// is exact 1:1, zero fee, both exact-in and exact-out (_getWrapInputRequired /
// _getUnwrapInputRequired are identity functions; the hooks differ only in
// approve-call plumbing (USDT) and ETH<->WETH conversion (ETH), not pricing).
//
// Source: https://docs.ring.exchange/contracts/v4/guides/fewtoken-liquidity-aggregation
// Manifest: https://docs.ring.exchange/assets/files/fewtoken-v4-pools-ethereum-e2c683d433aa86244404a1fdc329af79.json
// Cross-checked on-chain: hook/token bytecode present, and the DAI/fwDAI
// poolId (keccak256(abi.encode(PoolKey))) recomputed from the manifest's
// PoolKey matches the manifest's published poolId.
//
// See few/v1 for an older generation pool (WBTC/fwWBTC) not covered here.
var fewTokens = []few.TokenInfo{
	{
		// ETH/fwWETH: https://etherscan.io/address/0x7a5a8f5a36a6a2e9961caf6bb047a5a7580d0fe16a532aad93efc596028dfa54
		// Pool currency0 is the zero address (native ETH) on-chain; keyed here by WETH so
		// callers requesting a WETH<->fwWETH wrap resolve to this pool, settling in native ETH.
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0x7a5a8f5a36a6a2e9961caf6bb047a5a7580d0fe16a532aad93efc596028dfa54",
		HookAddress:        "0x044301939deb7ca53c4733dd4d9b3bc5ea0c6888",
		IsNative:           true,
		UnwrapTokenAddress: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		FewTokenAddress:    "0xa250cc729bb3323e7933022a67b52200fe354767", // fwWETH
		Fee:                0,
		TickSpacing:        1,
	},
	{
		// USDT/fwUSDT (FewUSDTHook): https://etherscan.io/address/0x7db868544c8f7f6ddb107c7749c94f03c9e0155f2138aef3f8a020e4a469d95a
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0x7db868544c8f7f6ddb107c7749c94f03c9e0155f2138aef3f8a020e4a469d95a",
		HookAddress:        "0xbadf77d50478b4432ef1f243b9c0bc7869486888",
		UnwrapTokenAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		FewTokenAddress:    "0xef87f4608e601e8564800265aee1c1ffadf73283", // fwUSDT
		Fee:                0,
		TickSpacing:        1,
	},
	{
		// USDC/fwUSDC: https://etherscan.io/address/0x5837e6b4fd4b8193f2f7a8b4490c0f154344bb9a52b36a885578ff6d3193fc47
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0x5837e6b4fd4b8193f2f7a8b4490c0f154344bb9a52b36a885578ff6d3193fc47",
		HookAddress:        "0x4b2eb653d13e6c9ac5a0a01fde22f2c8d6592888",
		UnwrapTokenAddress: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		FewTokenAddress:    "0x0492560fa7cfd6a85e50d8be3f77318994f8f429", // fwUSDC
		Fee:                0,
		TickSpacing:        1,
	},
	{
		// DAI/fwDAI: https://etherscan.io/address/0xf906beb74154ca4d057b7079c90eb1044efaf40ef468e62ec983930cf80a1e2b
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0xf906beb74154ca4d057b7079c90eb1044efaf40ef468e62ec983930cf80a1e2b",
		HookAddress:        "0x85b648a64aed6307d5d5ce26e6ae086c17bde888",
		UnwrapTokenAddress: "0x6b175474e89094c44da98b954eedeac495271d0f",
		FewTokenAddress:    "0x8a6fe57c08c84e0f4ee97aae68a62e820a37d259", // fwDAI
		Fee:                0,
		TickSpacing:        1,
	},
	{
		// WBTC/fwWBTC (current manifest pool): https://etherscan.io/address/0x18605c7a76101aeccc414cc300dd5e5ae44b30d6c247ba164ccd88952c259735
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0x18605c7a76101aeccc414cc300dd5e5ae44b30d6c247ba164ccd88952c259735",
		HookAddress:        "0x0fe942afdb2f51e25cbf892aad175c6a574f2888",
		UnwrapTokenAddress: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
		FewTokenAddress:    "0x2078f336fdd260f708bec4a20c82b063274e1b23", // fwWBTC
		Fee:                0,
		TickSpacing:        1,
	},
	{
		// cbBTC/fwcbBTC: https://etherscan.io/address/0x8f8b0b21fb429ffb5210f2bf0f8b7cb267b944a0c61beaae35f20f6839c0f33b
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0x8f8b0b21fb429ffb5210f2bf0f8b7cb267b944a0c61beaae35f20f6839c0f33b",
		HookAddress:        "0x8347b7a3807c681513d2b51b8223e59aa16a2888",
		UnwrapTokenAddress: "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf",
		FewTokenAddress:    "0xdbf1703e5d29afefbf1bd958ce7a6023c67f3e5d", // fwcbBTC
		Fee:                0,
		TickSpacing:        1,
	},
	{
		// weETH/fwweETH: https://etherscan.io/address/0x6933dfbf7441cc4ee4439843fdd464e215a6c90f07c5a769198e2a047f1f3f3e
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0x6933dfbf7441cc4ee4439843fdd464e215a6c90f07c5a769198e2a047f1f3f3e",
		HookAddress:        "0x877323adbf747f85eb8d182d42f01f34a5492888",
		UnwrapTokenAddress: "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
		FewTokenAddress:    "0x9553d5f1f564ede30f5a9f0274cd0af7a00546e7", // fwweETH
		Fee:                0,
		TickSpacing:        1,
	},
	{
		// wstETH/fwwstETH: https://etherscan.io/address/0xe7c2f30fd89238331b0e3e6ac6351578d5e3091b7839eff321c29cf88e17274e
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0xe7c2f30fd89238331b0e3e6ac6351578d5e3091b7839eff321c29cf88e17274e",
		HookAddress:        "0x75ae0292e8ad3ab60b9a1a7b3046d3f4abdfa888",
		UnwrapTokenAddress: "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
		FewTokenAddress:    "0xb90e63487bc6a4fa3d58f707510dab3c28a63137", // fwwstETH
		Fee:                0,
		TickSpacing:        1,
	},
	{
		// UNI/fwUNI: https://etherscan.io/address/0x301d41ff23b73b209ab2b1112f4effd0d8ff978ec29d743c1431463f84cbec24
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0x301d41ff23b73b209ab2b1112f4effd0d8ff978ec29d743c1431463f84cbec24",
		HookAddress:        "0x4b3e2a8cf36c7eb0fba2a5b39b20c896c6f22888",
		UnwrapTokenAddress: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
		FewTokenAddress:    "0xe8e1f50392bd61d0f8f48e8e7af51d3b8a52090a", // fwUNI
		Fee:                0,
		TickSpacing:        1,
	},
}

func NewTokenWrapper() few.TokenWrapper {
	return few.NewTokenWrapper(fewTokens)
}
