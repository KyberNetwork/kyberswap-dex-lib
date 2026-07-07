package renzo

import _ "embed"

//go:embed abis/Hook.json
var renzoHookABIJson []byte

//go:embed abis/RateProvider.json
var rateProviderABIJson []byte
