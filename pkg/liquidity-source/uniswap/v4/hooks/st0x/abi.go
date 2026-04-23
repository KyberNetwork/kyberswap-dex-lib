package st0x

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var (
	propAMMHookABI = lo.Must(abi.JSON(bytes.NewReader(propAMMHookABIJson)))
	priceOracleABI = lo.Must(abi.JSON(bytes.NewReader(priceOracleABIJson)))
)
