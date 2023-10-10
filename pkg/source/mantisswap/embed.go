package mantisswap

import _ "embed"

//go:embed abi/MainPool.json
var MainPoolABIBytes []byte

//go:embed abi/LP.json
var LPABIBytes []byte
