package stable

import _ "embed"

//go:embed abi/CLStableSwapHookFactory.json
var hookFactoryABIBytes []byte

//go:embed abi/CLStableSwapHook.json
var hookABIBytes []byte
