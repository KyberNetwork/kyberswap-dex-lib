package erc7575

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/ERC7575.json
var erc7575Json []byte

// erc7575ABI holds the ERC7575-specific share() getter that returns the (decoupled) share token address.
var erc7575ABI abi.ABI

const methodShare = "share"

func init() {
	var err error
	erc7575ABI, err = abi.JSON(bytes.NewReader(erc7575Json))
	if err != nil {
		panic(err)
	}
}
