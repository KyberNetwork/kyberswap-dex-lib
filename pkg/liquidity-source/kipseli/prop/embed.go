package prop

import _ "embed"

// KipseliPropLens is a deployless helper: its creation code runs as a
// `to`-less eth_call and reverts with a snapshot, never deployed. Source:
// KyberNetwork/ks-helper-sc src/helpers/deployless/KipseliPropLens.sol, built
// with `FOUNDRY_EVM_VERSION=paris forge build` (solc 0.8.25, 999999 runs);
// paris keeps PUSH0/MCOPY out so it runs on every chain.

//go:embed abis/KipseliPropLens.json
var lensABIData []byte

//go:embed abis/KipseliPropLens.bin
var lensBytecodeHex string
