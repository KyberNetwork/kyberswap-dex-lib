package nabla

import (
	_ "embed"
)

//go:embed abi/Portal.json
var portalBytes []byte

//go:embed abi/Router.json
var routerBytes []byte

//go:embed abi/SwapPool.json
var swapPoolBytes []byte

//go:embed abi/Curve.json
var curveBytes []byte

//go:embed abi/Oracle.json
var oracleBytes []byte
