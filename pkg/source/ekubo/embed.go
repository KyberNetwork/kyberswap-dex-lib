package ekubo

import _ "embed"

//go:embed abis/core.json
var coreJson []byte

//go:embed abis/data-fetcher.json
var dataFetcherJson []byte
