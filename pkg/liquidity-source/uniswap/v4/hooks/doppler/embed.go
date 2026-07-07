package doppler

import _ "embed"

//go:embed abis/UniswapV4ScheduledMulticurveInitializerHook.json
var hookABIJson []byte

//go:embed abis/DopplerHookInitializer.json
var poolStateABIJson []byte

//go:embed abis/RehypeDopplerHook.json
var rehypeDopplerHookABIJson []byte
