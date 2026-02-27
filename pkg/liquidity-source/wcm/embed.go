package wcm

import _ "embed"

//go:embed abis/CompositeExchange.json
var compositeExchangeJson []byte

//go:embed abis/SpotOrderBook.json
var spotOrderBookJson []byte
