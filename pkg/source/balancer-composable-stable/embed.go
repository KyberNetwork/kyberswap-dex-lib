package balancercomposablestable

import _ "embed"

//go:embed abis/Vault.json
var balancerVaultJson []byte

//go:embed abis/BalancerPool.json
var balancerPoolJson []byte

//go:embed abis/ComposableStable.json
var balancerComposableStablePoolJson []byte
