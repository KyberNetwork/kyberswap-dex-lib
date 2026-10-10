package apefun

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var hookABI, factoryABI abi.ABI

func init() {
	for _, a := range []struct {
		dst  *abi.ABI
		json []byte
	}{{&hookABI, hookABIJson}, {&factoryABI, factoryABIJson}} {
		var err error
		if *a.dst, err = abi.JSON(bytes.NewReader(a.json)); err != nil {
			panic(err)
		}
	}
}
