package nadfun

import (
	"github.com/holiman/uint256"
)

type Extra struct {
	IsLocked    bool `json:"isLocked"`    // Trading locked when target reached
	IsGraduated bool `json:"isGraduated"` // Token listed on Uniswap

	VirtualNative *uint256.Int `json:"virtualNative"`
	VirtualToken  *uint256.Int `json:"virtualToken"`

	K           *uint256.Int `json:"k"`
	TargetToken *uint256.Int `json:"targetToken"`

	ProtocolFee *uint256.Int `json:"protocolFee"`
}

type StaticExtra struct {
	Router string `json:"router"`
}

type SwapInfo struct {
	NewVirtualNative      *uint256.Int
	NewVirtualToken       *uint256.Int
	NewRealNativeReserves *uint256.Int
	NewRealTokenReserves  *uint256.Int
	IsLocked              bool
	IsBuy                 bool   `json:"isBuy"`
	Router                string `json:"router"`
}
