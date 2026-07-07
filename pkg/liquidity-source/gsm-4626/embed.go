package gsm4626

import _ "embed"

//go:embed abi/Gsm4626.json
var gsm4626Bytes []byte

//go:embed abi/PriceStrategy.json
var priceStrategyBytes []byte

//go:embed abi/FeeStrategy.json
var feeStrategyBytes []byte
