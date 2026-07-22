package machima

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI        abi.ABI
	clankNowABI    abi.ABI
	tokenABI       abi.ABI
	swapAdapterABI abi.ABI
)

func init() {
	for _, b := range []struct {
		abi  *abi.ABI
		data []byte
	}{
		{&poolABI, poolABIBytes},
		{&clankNowABI, clankNowABIBytes},
		{&tokenABI, tokenABIBytes},
		{&swapAdapterABI, swapAdapterABIBytes},
	} {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.abi = parsed
	}
}
