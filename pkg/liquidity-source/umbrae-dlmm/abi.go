package umbraedlmm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	pairABI    abi.ABI
	factoryABI abi.ABI
	viewerABI  abi.ABI
)

func init() {
	for _, b := range []struct {
		abi  *abi.ABI
		data []byte
	}{
		{&pairABI, pairABIJson},
		{&factoryABI, factoryABIJson},
		{&viewerABI, viewerABIJson},
	} {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.abi = parsed
	}
}
