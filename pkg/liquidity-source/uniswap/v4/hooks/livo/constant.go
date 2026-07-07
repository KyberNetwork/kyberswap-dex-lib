package livo

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	// LpFeeBps is the LP fee in basis points (1% = 100 bps)
	LpFeeBps = 100
)

var (
	h = common.HexToAddress

	HookAddresses = []common.Address{
		h("0x627FA6F76FA96b10BAe1B6Fba280A3c9264500Cc"), // Livo Hook
	}
)
