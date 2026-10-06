package flywheelfun

import (
	_ "embed"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abi/Factory.json
var factoryBytes string

//go:embed abi/Token.json
var tokenBytes string
var factoryABI = mustABI(factoryBytes)
var tokenABI = mustABI(tokenBytes)

func mustABI(s string) abi.ABI {
	a, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return a
}
