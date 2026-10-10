package apefun

import _ "embed"

//go:embed abis/LaunchFeeProtectionHook.json
var hookABIJson []byte

//go:embed abis/LaunchRegistryFactory.json
var factoryABIJson []byte
