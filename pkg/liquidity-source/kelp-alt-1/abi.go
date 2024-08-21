package rsethalt1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	RsETHPool       abi.ABI
	WstethETHOracle abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&RsETHPool, rsETHPool,
		}, {
			&WstethETHOracle, wstethETHOracle,
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
