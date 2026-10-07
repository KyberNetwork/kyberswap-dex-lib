package lglclob

import _ "embed"

//go:embed abis/OnchainClob.json
var OnchainClobABIJson []byte

// Lens is a deployless helper, abis/Lens.sol built by solc 0.8.25 for shanghai
// with via-ir and no metadata. Its constructor reads the arguments abis/Lens.json
// declares from the end of its own code, rather than taking them, as that keeps
// the code sent with every call small.
//
//go:embed abis/Lens.json
var lensABIJson []byte

//go:embed abis/Lens.bin
var lensBytecodeHex string
