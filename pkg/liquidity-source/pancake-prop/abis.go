package pancakeprop

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	//go:embed abis/Entry.json
	entryABIData []byte
	//go:embed abis/Pricing.json
	pricingABIData []byte

	entryABI   = mustABI(entryABIData)
	pricingABI = mustABI(pricingABIData)
)

func mustABI(raw []byte) abi.ABI {
	parsed, err := abi.JSON(bytes.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}
