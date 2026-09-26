package spireprop

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	//go:embed abis/Entrypoint.json
	entrypointABIData []byte
	//go:embed abis/CurveBook.json
	curveBookABIData []byte
	//go:embed abis/Custodian.json
	custodianABIData []byte
	entrypointABI    = mustABI(entrypointABIData)
	curveBookABI     = mustABI(curveBookABIData)
	custodianABI     = mustABI(custodianABIData)
)

func mustABI(raw []byte) abi.ABI {
	parsed, err := abi.JSON(bytes.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}
