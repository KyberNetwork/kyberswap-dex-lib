package eulerswap

import (
	"math/big"

	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Vault0              string       `json:"v0"`
	Vault1              string       `json:"v1"`
	EulerAccount        string       `json:"ea"`
	FeeMultiplier       *uint256.Int `json:"fm"`
	EquilibriumReserve0 *uint256.Int `json:"er0"`
	EquilibriumReserve1 *uint256.Int `json:"er1"`
	PriceX              *uint256.Int `json:"px"`
	PriceY              *uint256.Int `json:"py"`
	ConcentrationX      *uint256.Int `json:"cx"`
	ConcentrationY      *uint256.Int `json:"cy"`
	Pause               bool         `json:"p"`
}

type Extra struct {
	Pause  uint32  `json:"p"`
	Vaults []Vault `json:"v"`
}

type SwapInfo struct {
	NewReserve0 *uint256.Int `json:"newReserve0"`
	NewReserve1 *uint256.Int `json:"newReserve1"`
}

type Vault struct {
	Cash               *uint256.Int
	Debt               *uint256.Int
	MaxDeposit         *uint256.Int
	MaxWithdraw        *uint256.Int
	TotalBorrows       *uint256.Int
	EulerAccountAssets *uint256.Int
}

type ReserveRPC struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
	Pause    uint32
}

type VaultRPC struct {
	Cash                *big.Int
	Debt                *big.Int
	MaxDeposit          *big.Int
	MaxWithdraw         *big.Int
	TotalBorrows        *big.Int
	EulerAccountBalance *big.Int
	TotalAssets         *big.Int
	TotalSupply         *big.Int
}

type TrackerData struct {
	Vaults   []VaultRPC
	Reserves ReserveRPC
}
