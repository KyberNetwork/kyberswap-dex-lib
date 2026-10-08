package uscoreprop

import _ "embed"

//go:embed abi/USCorePool.json
var poolABIJson []byte

// USCoreSampler is a deployless helper: abi/USCoreSampler.sol built with solc 0.8.30
// (--via-ir --optimize --optimize-runs 1 --evm-version paris --no-cbor-metadata), so it runs
// without PUSH0/MCOPY. Its creation code runs as a to-less eth_call and reverts with Sample.
//
//go:embed abi/USCoreSampler.json
var samplerABIJson []byte

//go:embed abi/USCoreSampler.bin
var samplerBytecodeHex string
