package integral

import "net/http"

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`

	AlwaysUseTickLens bool
	TickLensAddress   string

	UseBasePluginV2 bool `json:"useBasePluginV2"`

	// SkipBlockPinning stops the plugin reads being pinned to the block number
	// that the first multicall reported. Multicall3 returns the EVM's
	// block.number, which on most chains equals the RPC block height — but not
	// on all of them. Robinhood Chain (4663) is one where it does not: at the
	// time of writing eth_blockNumber is 37,204,546 while block.number is
	// 25,761,193, and the latter does not advance in step (0 vs 70 blocks over
	// the same 6 seconds). Feeding that value back as an eth_call block tag asks
	// the node for state ~11.4M blocks in its past, which it answers with
	// "metadata is not found" and the whole refresh fails.
	//
	// Leave false to keep pinning, which is what every currently-supported chain
	// wants: it keeps a multi-call refresh consistent on one block.
	SkipBlockPinning bool `json:"skipBlockPinning"`
}
