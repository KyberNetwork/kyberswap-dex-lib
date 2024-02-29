package bancorv21

import "errors"

const (
	PPM_RESOLUTION = 1000000

	getAnchorCount             = "getAnchorCount"
	registryGetAnchors         = "getAnchors"
	getConvertersByAnchors     = "getConvertersByAnchors"
	getConvertibleTokenAnchors = "getConvertibleTokenAnchors"
	getConvertibleTokens       = "getConvertibleTokens"
	DexType                    = "bancor-v21"
	DexTypeBancorV21InnerPool  = "bancor-v21-inner-pool"

	reserveZero            = "0"
	converterGetTokenCount = "connectorTokenCount"
	converterGetTokens     = "connectorTokens"
	converterGetReserve    = "getConnectorBalance"
	converterGetFee        = "conversionFee"
	// BancorTokenAddress bnt anchor token as anchor token for path finder
	BancorTokenAddress = "0x1F573D6Fb3F13d689FF844B4cE37794d79a7FF1C"
)

var (
	defaultGas                   = Gas{Swap: 60000}
	ErrPairAddressNotMatchAnchor = errors.New("pair address not match anchor")
	ErrInvalidToken              = errors.New("invalid token")
	ErrInvalidPath               = errors.New("invalid inner anchor path")
	ErrInvalidAnchor             = errors.New("invalid anchor")
)
