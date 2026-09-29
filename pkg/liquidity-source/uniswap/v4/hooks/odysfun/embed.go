package odysfun

import _ "embed"

//go:embed abis/OdysHook.json
var odysHookABIJson []byte

//go:embed abis/OdysHookLegacy.json
var odysHookLegacyABIJson []byte
