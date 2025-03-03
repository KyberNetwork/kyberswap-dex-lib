package abis

import _ "embed"

//go:embed MetaAggregationRouterV2.json
var metaAggregationRouterV2 []byte

//go:embed ERC20.json
var erc20 []byte

//go:embed MetaAggregationRouterV2Optimistic.json
var metaAggregationRouterV2Optimistic []byte

//go:embed ScrolL1GasPriceOracle.json
var scrolL1GasPriceOracle []byte

//go:embed OptimismGasPriceOracle.json
var optimismGasPriceOracle []byte

//go:embed ArbGasInfo.json
var arbGasInfo []byte
