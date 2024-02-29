package bancorv21

import _ "embed"

//go:embed abis/converter_registry.json
var converterRegistryJson []byte

//go:embed abis/converter_v23.json
var converterJson []byte
