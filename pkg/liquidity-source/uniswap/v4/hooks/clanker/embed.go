package clanker

import _ "embed"

//go:embed abis/Clanker.json
var clankerABIJson []byte

//go:embed abis/ClankerHookDynamicFee.json
var dynamicFeeHookABIJson []byte

//go:embed abis/ClankerHookStaticFee.json
var staticFeeHookABIJson []byte
