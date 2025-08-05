package bunniv2

import _ "embed"

//go:embed abis/BunniHub.json
var bunniHubABIJson []byte

//go:embed abis/BunniHook.json
var bunniHookABIJson []byte

//go:embed abis/FeeOverrideHooklet.json
var feeOverrideHookletABIJson []byte

//go:embed abis/ERC4626.json
var erc4626ABIJson []byte
