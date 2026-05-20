package uniswapv3

import (
	_ "embed"
)

//go:embed pregenesispools/optimism.json
var optimismPreGenesisPoolsBytes []byte

//go:embed pregenesispools/rubicon-clmm-mainnet.json
var rubiconClmmMainnetPreGenesisPoolsBytes []byte

//go:embed pregenesispools/rubicon-clmm-optimism.json
var rubiconClmmOptimismPreGenesisPoolsBytes []byte

//go:embed pregenesispools/rubicon-clmm-arbitrum.json
var rubiconClmmArbitrumPreGenesisPoolsBytes []byte

//go:embed pregenesispools/rubicon-clmm-base.json
var rubiconClmmBasePreGenesisPoolsBytes []byte

var BytesByPath = map[string][]byte{
	"pregenesispools/optimism.json":              optimismPreGenesisPoolsBytes,
	"pregenesispools/rubicon-clmm-mainnet.json":  rubiconClmmMainnetPreGenesisPoolsBytes,
	"pregenesispools/rubicon-clmm-optimism.json": rubiconClmmOptimismPreGenesisPoolsBytes,
	"pregenesispools/rubicon-clmm-arbitrum.json": rubiconClmmArbitrumPreGenesisPoolsBytes,
	"pregenesispools/rubicon-clmm-base.json":     rubiconClmmBasePreGenesisPoolsBytes,
}
