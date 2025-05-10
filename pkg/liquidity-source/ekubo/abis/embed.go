package abis

import _ "embed"

//go:embed Core.json
var coreJson []byte

//go:embed Twamm.json
var twammJson []byte

//go:embed BasicDataFetcher.json
var basicDataFetcherJson []byte

//go:embed TwammDataFetcher.json
var twammDataFetcherJson []byte
