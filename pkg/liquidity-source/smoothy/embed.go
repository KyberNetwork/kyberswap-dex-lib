package smoothy

import (
	_ "embed"
)

//go:embed abi/SmoothyV1.json
var smoothyV1ABIBytes []byte
