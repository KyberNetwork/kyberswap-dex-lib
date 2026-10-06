package kuruobv2

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/KuruV2.json
var kuruV2ABIJson []byte

// kuruV2ABI holds the OrderBook views plus AccountCore.protocolPaused.
var kuruV2ABI = func() abi.ABI {
	a, err := abi.JSON(bytes.NewReader(kuruV2ABIJson))
	if err != nil {
		panic(err)
	}
	return a
}()
