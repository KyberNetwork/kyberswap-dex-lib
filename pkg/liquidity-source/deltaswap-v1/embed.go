package deltaswapv1

import _ "embed"

//go:embed abis/DeltaSwapV1Factory.json
var deltaSwapV1FactoryABIJson []byte

//go:embed abis/DeltaSwapV1Pair.json
var DeltaSwapV1PairABIJson []byte
