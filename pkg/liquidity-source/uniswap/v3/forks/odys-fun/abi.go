package odysfun

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/OdysToken.json
var odysTokenABIJson []byte

var odysTokenABI abi.ABI

func init() {
	var err error
	if odysTokenABI, err = abi.JSON(bytes.NewReader(odysTokenABIJson)); err != nil {
		panic(err)
	}
}
