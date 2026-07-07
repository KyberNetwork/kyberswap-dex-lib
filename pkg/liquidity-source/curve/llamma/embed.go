package llamma

import _ "embed"

//go:embed abi/CurveControllerFactory.json
var curveControllerFactoryABIBytes []byte

//go:embed abi/CurveLlamma.json
var curveLlammaABIBytes []byte
