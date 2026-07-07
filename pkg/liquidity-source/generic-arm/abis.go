package genericarm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	lidoArmABI abi.ABI
	ERC626ABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&lidoArmABI, lidoArmABIData},
		{&ERC626ABI, ERC626ABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
