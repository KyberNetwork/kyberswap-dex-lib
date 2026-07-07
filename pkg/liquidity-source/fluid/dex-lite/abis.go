package dexLite

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

// Parsed ABI instances
var (
	fluidDexLiteABI = lo.Must(abi.JSON(strings.NewReader(string(fluidDexLiteABIBytes))))
	centerPriceABI  = lo.Must(abi.JSON(strings.NewReader(string(centerPriceABIBytes))))
)
