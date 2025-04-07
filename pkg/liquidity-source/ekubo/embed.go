package ekubo

import _ "embed"

//go:embed abis/Core.json
var coreJson []byte

//go:embed abis/DataFetcher.json
var dataFetcherJson []byte
