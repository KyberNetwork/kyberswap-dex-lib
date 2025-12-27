package cusd

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	capTokenABI            abi.ABI
	oracleABI              abi.ABI
	pausableUpgradeableABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&capTokenABI, capTokenBytes},
		{&oracleABI, oracleBytes},
		{&pausableUpgradeableABI, pausableUpgradeableBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
