package atokenswap

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

// Parsed ABI instance
var (
	aTokenSwapABI = lo.Must(abi.JSON(strings.NewReader(string(aTokenSwapABIBytes))))
)
