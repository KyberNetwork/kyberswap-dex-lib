package volatile

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var volatilePairHookABI abi.ABI

func init() {
	var err error
	volatilePairHookABI, err = abi.JSON(bytes.NewReader(volatilePairHookABIJson))
	if err != nil {
		panic(err)
	}
}
