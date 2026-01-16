package hooks

import (
	_ "embed"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/EulerSwapHook.json
var hookABIJson []byte

var HookABI abi.ABI

func init() {
	var err error
	HookABI, err = abi.JSON(strings.NewReader(string(hookABIJson)))
	if err != nil {
		panic(err)
	}
}
