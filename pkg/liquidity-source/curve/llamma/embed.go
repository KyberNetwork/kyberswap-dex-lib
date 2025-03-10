package llamma

import _ "embed"

//go:embed abi/Llamma.json
var llammaABIBytes []byte

//go:embed abi/Factory.json
var factoryABIBytes []byte

//go:embed abi/CurveLlammaHelper.json
var llammaHelperABIBytes []byte
