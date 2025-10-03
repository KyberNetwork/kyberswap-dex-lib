package cusd

import (
	_ "embed"
)

//go:embed abi/CapToken.json
var capTokenBytes []byte

//go:embed abi/Oracle.json
var oracleBytes []byte
