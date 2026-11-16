package btrprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

// poolABI is the embedded abis/Pool.json, a pinned copy of the deployed BTR Pool ABI.
var poolABI = lo.Must(abi.JSON(bytes.NewReader(poolABIJSONBytes)))

// hookABI is the read surface of BTR's yield hooks (dex-evm src/hooks): the shared venueStatus
// view and one getter per venue adapter, used to identify the hook's kind.
var hookABI = lo.Must(abi.JSON(bytes.NewReader(hookABIJSONBytes)))
