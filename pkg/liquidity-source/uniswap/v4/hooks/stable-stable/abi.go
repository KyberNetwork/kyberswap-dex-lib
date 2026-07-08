package stablestable

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var stableStableHookABI = lo.Must(abi.JSON(bytes.NewReader(stableStableHookABIJson)))
