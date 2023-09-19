package abis

import _ "embed"

//go:embed MetaAggregationRouterV2.json
var metaAggregationRouterV2 []byte

//go:embed ERC20.json
var erc20 []byte

//go:embed MetaAggregationRouterV2Optimistic.json
var metaAggregationRouterV2Optimistic []byte
