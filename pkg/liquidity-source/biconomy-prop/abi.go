package biconomyprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var venueABI abi.ABI

func init() {
	var err error
	venueABI, err = abi.JSON(bytes.NewReader(venueABIJson))
	if err != nil {
		panic(err)
	}
}
