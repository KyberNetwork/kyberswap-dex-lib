package syntropiaprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI abi.ABI
	lensABI abi.ABI
)

func init() {
	for _, b := range []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&poolABI, poolABIData},
		{&lensABI, lensABIData},
	} {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
