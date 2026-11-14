package peach

import (
	_ "embed"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/Hook.json
var hookABIJSON string

var hookABI = mustABI(hookABIJSON)
var clockABI = mustABI(`[{"type":"function","name":"getCurrentBlockTimestamp","inputs":[],"outputs":[{"name":"timestamp","type":"uint256"}],"stateMutability":"view"}]`)

func mustABI(source string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(source))
	if err != nil {
		panic(err)
	}
	return parsed
}
