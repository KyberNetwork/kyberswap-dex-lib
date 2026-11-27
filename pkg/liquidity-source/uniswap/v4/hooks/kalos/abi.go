package kalos

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var kalosDynamicABI, poolManagerABI abi.ABI

func init() {
	for _, a := range []struct {
		abi  *abi.ABI
		json []byte
	}{{&kalosDynamicABI, kalosDynamicABIJson}, {&poolManagerABI, poolManagerABIJson}} {
		var err error
		if *a.abi, err = abi.JSON(bytes.NewReader(a.json)); err != nil {
			panic(err)
		}
	}
}
