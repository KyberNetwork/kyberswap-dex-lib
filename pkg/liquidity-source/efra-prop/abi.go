package efraprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// abiJSON is the subset of PropAmmContract's ABI this package calls: USDC() for the quote
// token, quoteExactIn() for the only pricing source (the venue's curve is not replicated
// off-chain -- see pool_tracker.go).
var (
	venueABI abi.ABI
)

func init() {
	parsed, err := abi.JSON(bytes.NewReader(abiJSON))
	if err != nil {
		panic(err)
	}
	venueABI = parsed
}
