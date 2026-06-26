package fourmeme

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	//go:embed abis/TokenTax.json
	tokenTaxABIJSON []byte
	tokenTaxABI     abi.ABI
)

func init() {
	var err error
	tokenTaxABI, err = abi.JSON(bytes.NewReader(tokenTaxABIJSON))
	if err != nil {
		panic(err)
	}
}
