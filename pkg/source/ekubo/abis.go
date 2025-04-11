package ekubo

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var coreABI, dataFetcherABI abi.ABI
var positionUpdatedEvent abi.Event

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&coreABI, coreJson},
		{&dataFetcherABI, dataFetcherJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	positionUpdatedEvent = coreABI.Events["PositionUpdated"]
}
