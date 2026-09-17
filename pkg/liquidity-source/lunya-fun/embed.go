package lunyafun

import _ "embed"

//go:embed abis/LunyaLaunch.json
var launchABIBytes []byte

//go:embed abis/LunyaLaunchFactory.json
var factoryABIBytes []byte
