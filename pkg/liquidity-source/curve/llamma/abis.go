package llamma

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	curveControllerFactoryABI abi.ABI
	curveLlammaABI            abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&curveControllerFactoryABI, curveControllerFactoryABIBytes},
		{&curveLlammaABI, curveLlammaABIBytes},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
