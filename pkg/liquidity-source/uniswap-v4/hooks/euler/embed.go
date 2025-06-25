package euler

import _ "embed"

//go:embed abis/EulerSwap.json
var poolABIJson []byte

//go:embed abis/EulerSwapFactory.json
var factoryABIJson []byte

//go:embed abis/EVault.json
var vaultABIJson []byte

//go:embed abis/EthereumVaultConnector.json
var evcABIJson []byte
