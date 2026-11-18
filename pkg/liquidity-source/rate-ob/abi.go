package rateob

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/Rate.json
var rateABIJson []byte

// rateABI holds the Orderbook, MatchingEngine and OrderbookFactory views the source reads.
var rateABI = func() abi.ABI {
	a, err := abi.JSON(bytes.NewReader(rateABIJson))
	if err != nil {
		panic(err)
	}
	return a
}()
