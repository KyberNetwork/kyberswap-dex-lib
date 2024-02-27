package bancor_v21

const (
	getAnchorCount             = "getAnchorCount"
	registryGetAnchors         = "getAnchors"
	registryGetAnchor          = "getAnchor"
	getConvertersByAnchors     = "getConvertersByAnchors"
	getConvertibleTokenAnchors = "getConvertibleTokenAnchors"
	getConvertibleTokens       = "getConvertibleTokens"
	DexTypeBancorV21           = "bancor-v21"
	DexTypeBancorV21InnerPool  = "bancor-v21-inner-pool"

	reserveZero            = "0"
	PPM_RESOLUTION         = 1000000
	converterGetTokenCount = "connectorTokenCount"
	converterGetTokens     = "connectorTokens"
	converterGetReserve    = "getConnectorBalance"
	converterGetFee        = "conversionFee"
)

var defaultGas = Gas{Swap: 60000}
