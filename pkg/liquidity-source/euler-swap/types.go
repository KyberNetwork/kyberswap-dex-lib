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
	ZeroForOne  bool         `json:"zeroForOne"`
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
	TotalBorrows        *big.Int
	EulerAccountBalance *big.Int
	TotalAssets         *big.Int
	TotalSupply         *big.Int
	Caps                [2]uint16
	MaxWithdraw         *big.Int
}

type TrackerData struct {
	Vaults   []VaultRPC
	Reserves ReserveRPC
}

type PoolExtra struct {
	Vault0              string   `json:"vault0"`
	Vault1              string   `json:"vault1"`
	EulerAccount        string   `json:"eulerAccount"`
	EquilibriumReserve0 *big.Int `json:"equilibriumReserve0"`
	EquilibriumReserve1 *big.Int `json:"equilibriumReserve1"`
	FeeMultiplier       *big.Int `json:"feeMultiplier"`
	PriceY              *big.Int `json:"priceY"`
	PriceX              *big.Int `json:"priceX"`
	ConcentrationY      *big.Int `json:"concentrationY"`
	ConcentrationX      *big.Int `json:"concentrationX"`
	BlockNumber         uint64   `json:"blockNumber"`
}
