package frxusd

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	FrxUsdCustodianUsdcABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&FrxUsdCustodianUsdcABI, frxUsdCustodianUsdcBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
