package stake

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var StakeManagerABI abi.ABI

func init() {
	var err error
	StakeManagerABI, err = abi.JSON(bytes.NewReader(stakeManagerABIData))
	if err != nil {
		panic(err)
	}
}
