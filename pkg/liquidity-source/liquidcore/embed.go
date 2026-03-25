package liquidcore

import _ "embed"

//go:embed abis/Pool.json
var poolBytes []byte

//go:embed abis/Router.json
var routerBytes []byte
