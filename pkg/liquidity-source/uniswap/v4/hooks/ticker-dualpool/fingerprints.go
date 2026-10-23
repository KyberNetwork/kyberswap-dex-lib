package tickerdualpool

import "github.com/ethereum/go-ethereum/common"

// Generated from the pinned Ticker compiler artifacts. These identify accounting
// implementations; deployment authorization remains a separate address decision.
var tickerHookFingerprint = runtimeFingerprint{Size: 37586, Hash: common.HexToHash("0x8874031706e3284e478eae1468a9e6940d506a91d966cdad628b5cce2ebad76c"), ImmutableOffsets: []int{2416, 3571, 5131, 5197, 5791, 5943, 6105, 6600, 7258, 7826, 8207, 8689, 8830, 9456, 9563, 9716, 10588, 11047, 11328, 11396, 12096, 12194, 14796, 17335, 17699, 21466, 22614, 23274, 24764, 25199, 25268, 26026, 26569, 27661, 28351, 28502, 32382, 32991, 34998}}
var tickerVaultFingerprint = runtimeFingerprint{Size: 12044, Hash: common.HexToHash("0xc34334e77e1c8bb3556719f884249d7c41f926a0469b850052fc29dde3fb53b7"), ImmutableOffsets: []int{503, 892, 1527, 2426, 2933, 3803, 4210, 4486, 5566, 5846, 6421, 6577, 6646, 7524, 7602, 7931, 8447, 9257, 9510, 9772, 9953, 10146, 10375, 10840, 11119}}

// Same ERC1967Proxy executable in the original fixture and staged deployment.
// Compiler metadata changes with source paths; the delegatecall/slot instructions must not.
var tickerProxyFingerprint = runtimeFingerprint{Size: 77, Hash: common.HexToHash("0x3c04593ddc0d8247a804e8f554bee9d2e505540d3b0d4a567d4ff589c25498a6")}

// Monad implementation 0x844A14BaE1B67DF6064bB87BA5Ff941DC02649e0.
// Its executable differs from the original fingerprint only in the three
// SWEEPING_SLOT constants and the PoolBuybackBurned event topic. Replacing those
// with the old slot/topic reproduces tickerHookFingerprint exactly; swap math,
// storage layout and immutable offsets are unchanged.
var tickerMainnetHookFingerprint = runtimeFingerprint{
	Size:             tickerHookFingerprint.Size,
	Hash:             common.HexToHash("0x914b298a03e9ba9f6d28db4817d51a93353c969b5d543deb4a0426b26df1b57c"),
	ImmutableOffsets: tickerHookFingerprint.ImmutableOffsets,
}

func matchesTickerHook(code []byte) bool {
	return tickerHookFingerprint.matches(code) || tickerMainnetHookFingerprint.matches(code)
}
