package mkr_sky

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	mkrSkyABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&mkrSkyABI, mkrSkyABIData,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
