package ghost

import _ "embed"

//go:embed abis/OffchainQuotedLinearFee.json
var feeABIData []byte

//go:embed abis/CrossCollateralRouter.json
var routerABIData []byte

//go:embed abis/RoutingFee.json
var routingFeeABIData []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed pools/arbitrum.json
var arbitrumPoolData []byte

//go:embed pools/base.json
var basePoolData []byte

var bytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
	"pools/arbitrum.json": arbitrumPoolData,
	"pools/base.json":     basePoolData,
}
