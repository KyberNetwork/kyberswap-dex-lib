package common

import _ "embed"

//go:embed abis/swETH.json
var swETHABIJson []byte

//go:embed abis/rswETH.json
var rswETHABIJson []byte

//go:embed abis/AccessControlManager.json
var accessControlManagerABIJson []byte
