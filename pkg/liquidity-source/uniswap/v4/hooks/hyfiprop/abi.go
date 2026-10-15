package hyfiprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

// hyfiABI merges the hook's views (pairConfig, getBookSideRaw) and ERC20 balanceOf.
var hyfiABI = lo.Must(abi.JSON(bytes.NewReader(hyfiABIJson)))
