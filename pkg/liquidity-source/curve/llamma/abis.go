package llamma

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	CurveControllerFactoryABI abi.ABI
	CurveLlammaABI            abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&CurveControllerFactoryABI, curveControllerFactoryABIBytes},
		{&CurveLlammaABI, curveLlammaABIBytes},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
