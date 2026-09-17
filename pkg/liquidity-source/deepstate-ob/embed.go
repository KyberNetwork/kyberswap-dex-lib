package deepstateob

import _ "embed"

//go:embed abi/DeepstateV1.json
var deepstateV1ABIJson []byte

//go:embed abi/DeepstateBookLens.json
var deepstateBookLensABIJson []byte
