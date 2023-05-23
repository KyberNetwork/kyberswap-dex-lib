package balancer

import _ "embed"

//go:embed abis/Vault.json
var balancerVaultJson []byte

//go:embed abis/BalancerPool.json
var balancerPoolJson []byte

//go:embed abis/StablePool.json
var balancerStablePoolJson []byte

//go:embed abis/MetaStablePool.json
var balancerMetaStablePoolJson []byte
