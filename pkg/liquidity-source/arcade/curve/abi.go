package curve

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	arcadeHookABI abi.ABI

	launchCreatedEventHash common.Hash
	curveBuyEventHash      common.Hash
	curveSellEventHash     common.Hash
	graduatedEventHash     common.Hash
)

func init() {
	var err error
	arcadeHookABI, err = abi.JSON(bytes.NewReader(arcadeHookABIBytes))
	if err != nil {
		panic(err)
	}
	launchCreatedEventHash = arcadeHookABI.Events["LaunchCreated"].ID
	curveBuyEventHash = arcadeHookABI.Events["CurveBuy"].ID
	curveSellEventHash = arcadeHookABI.Events["CurveSell"].ID
	graduatedEventHash = arcadeHookABI.Events["Graduated"].ID
}
