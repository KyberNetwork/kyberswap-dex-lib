package makerpsm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	makerPSMPSM abi.ABI
	makerPSMVat abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&makerPSMPSM, makerPSMPSMBytes},
		{&makerPSMVat, makerPSMVatBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
