package mhocdesk

import _ "embed"

//go:embed abis/PropAMM.json
var propAMMABIData []byte

//go:embed abis/PropAMMRegistry.json
var registryABIData []byte
