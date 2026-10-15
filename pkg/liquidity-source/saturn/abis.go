package saturn

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	swapFacilityABI abi.ABI
	usdatABI        abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&swapFacilityABI, swapFacilityABIJson},
		{&usdatABI, usdatABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
