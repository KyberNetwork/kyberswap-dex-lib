package uscoreprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var poolABI = lo.Must(abi.JSON(bytes.NewReader(poolABIJson)))
