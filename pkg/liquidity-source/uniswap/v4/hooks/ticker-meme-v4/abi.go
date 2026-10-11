package tickermemev4

import (
	_ "embed"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/TickerMemeHook.json
var hookABIJSON string

var hookABI = func() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(hookABIJSON))
	if err != nil {
		panic(err)
	}
	return parsed
}()
