package liquidcore

import _ "embed"

//go:embed abi/Pool.json
var poolBytes []byte

//go:embed abi/Router.json
var routerBytes []byte
