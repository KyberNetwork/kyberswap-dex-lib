package btr

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

// poolABI is the embedded abis/Pool.json, a pinned copy of the deployed BTR Pool ABI.
var poolABI = lo.Must(abi.JSON(bytes.NewReader(poolABIJSONBytes)))
