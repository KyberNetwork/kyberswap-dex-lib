package altfun

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	pairABI           abi.ABI
	bondingABI        abi.ABI
	factoryABI        abi.ABI
	leveragedTokenABI abi.ABI
	globalStorageABI  abi.ABI
	zapABI            abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&pairABI, pairJSON},
		{&bondingABI, bondingJSON},
		{&factoryABI, factoryJSON},
		{&leveragedTokenABI, leveragedTokenJSON},
		{&globalStorageABI, globalStorageJSON},
		{&zapABI, zapJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
