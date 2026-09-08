package prmfun

import _ "embed"

//go:embed abis/MemeCurve.json
var memeCurveABIJson []byte

//go:embed abis/MemeFactory.json
var memeFactoryABIJson []byte

//go:embed abis/PremiumRouter.json
var premiumRouterABIJson []byte
