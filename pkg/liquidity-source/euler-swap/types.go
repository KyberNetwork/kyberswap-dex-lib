package eulerswap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
}

type SwapInfo struct {
	NewReserve0 *uint256.Int `json:"NewReserve0"`
	NewReserve1 *uint256.Int `json:"NewReserve1"`
}

type Vault struct {
	Cash         *uint256.Int
	Debt         *uint256.Int
	MaxDeposit   *uint256.Int
	MaxWithdraw  *uint256.Int
	TotalBorrows *uint256.Int
	Balance      *uint256.Int
}

type ReserveRPC struct {
	Reserve0       *big.Int
	Reserve1       *big.Int
	BlockTimestamp uint32
}

type VaultRPC struct {
	Cash         *big.Int
	Debt         *big.Int
	MaxDeposit   *big.Int
	MaxWithdraw  *big.Int
	TotalBorrows *big.Int
	Balance      *big.Int
}

type RPCData struct {
	Vault0              Vault
	Vault1              Vault
	EulerAccount        common.Address
	FeeMultiplier       *uint256.Int
	Reserve0            *uint256.Int
	Reserve1            *uint256.Int
	EquilibriumReserve0 *uint256.Int
	EquilibriumReserve1 *uint256.Int
	PriceX              *uint256.Int
	PriceY              *uint256.Int
	ConcentrationX      *uint256.Int
	ConcentrationY      *uint256.Int
}
