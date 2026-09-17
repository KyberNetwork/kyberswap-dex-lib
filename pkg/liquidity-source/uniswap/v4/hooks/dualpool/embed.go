package dualpool

import _ "embed"

//go:embed abis/DualPoolHook.json
var dualPoolHookABIJson []byte

//go:embed abis/StateView.json
var stateViewABIJson []byte
