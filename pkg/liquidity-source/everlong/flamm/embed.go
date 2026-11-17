package everlongflamm

import _ "embed"

//go:embed abi/Pool.json
var poolABIJson []byte

//go:embed abi/Factory.json
var factoryABIJson []byte

//go:embed abi/Router.json
var routerABIJson []byte

//go:embed abi/Hook.json
var hookABIJson []byte

//go:embed abi/Account.json
var accountABIJson []byte

// FlammSampler (abi/FlammSampler.sol) is never deployed: a node runs it by an eth_call state override at the taker,
// so the pool sees the caller a route's swap has. testdata/build_sampler.py builds it with solc 0.8.35 (standard
// JSON: viaIR, optimizer runs 1, evmVersion paris, no CBOR metadata), so it has no PUSH0 or MCOPY, and checks the
// committed files with --check. abi/FlammSampler.bin is its runtime code, in hex.
//
//go:embed abi/FlammSampler.json
var samplerABIJson []byte

//go:embed abi/FlammSampler.bin
var samplerRuntimeHex string
