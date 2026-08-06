package everlongcollvault

import _ "embed"

//go:embed abi/Rebalancer.json
var rebalancerABIJson []byte

//go:embed abi/Swapper.json
var swapperABIJson []byte

//go:embed abi/ALMAdapter.json
var almABIJson []byte

//go:embed abi/CollVault.json
var collVaultABIJson []byte
