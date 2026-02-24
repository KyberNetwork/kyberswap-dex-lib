package shared

import (
	"math/big"

	"github.com/holiman/uint256"
)

type VaultInfo struct {
	VaultAddress string
	AssetAddress string
	QuoteAmount  *big.Int
}

type ReserveRPC struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
	Status   uint32
}

type VaultRPC struct {
	Cash                *big.Int
	Debt                *big.Int
	MaxDeposit          *big.Int
	TotalBorrows        *big.Int
	TotalAssets         *big.Int
	TotalSupply         *big.Int
	EulerAccountBalance *big.Int
	MaxWithdraw         *big.Int // V1 only
	Caps                [2]uint16
	IsControllerEnabled bool
}

type AccountLiquidityRPC struct {
	CollateralValue *big.Int
	LiabilityValue  *big.Int
}

type VaultState struct {
	Cash                *uint256.Int    `json:"c,omitempty"`
	Debt                *uint256.Int    `json:"d,omitempty"` // debt of euler account
	MaxDeposit          *uint256.Int    `json:"mD,omitempty"`
	MaxWithdraw         *uint256.Int    `json:"mW,omitempty"` // V1 only
	TotalBorrows        *uint256.Int    `json:"tB,omitempty"`
	EulerAccountAssets  *uint256.Int    `json:"eAA,omitempty"` // converted assets balance
	BorrowCap           *uint256.Int    `json:"bC,omitempty"`  // V2 only
	DebtPrice           *uint256.Int    `json:"dP,omitempty"`  // for solvency check
	ValuePrices         []*uint256.Int  `json:"vP,omitempty"`  // collateral value prices
	VaultValuePrices    [2]*uint256.Int `json:"vVP,omitempty"` // vault value prices
	LTVs                []uint64        `json:"ltv,omitempty"`
	VaultLTVs           [2]uint64       `json:"vLtv,omitempty"`
	IsControllerEnabled bool            `json:"iCE,omitempty"`
}

type SwapInfo struct {
	Reserves              [2]*uint256.Int
	WithdrawAmount        *uint256.Int
	BorrowAmount          *uint256.Int
	ReserveDepositAmount  *uint256.Int
	VaultDepositAmount    *uint256.Int
	RepayAmount           *uint256.Int
	Debt                  *uint256.Int
	DebtVaultIdx          int
	CollateralValue       *uint256.Int
	IsSellVaultControlled bool
	IsBuyVaultControlled  bool
	ZeroForOne            bool `json:"0f1"`
}

type TrackerData struct {
	Vaults               []VaultRPC
	Reserves             ReserveRPC
	Controller           string            // controller debt vault, if exist
	VaultPrices          [3][3][2]*big.Int // other vault -> debt vault -> [bid/value,ask/debt]
	VaultLtvs            [3][3]uint16      // vault 0/1/controller -> debt vault
	CollatAmounts        []*big.Int        // asset amount of euler account across collateral vaults
	CollatPrices         [][3][2]*big.Int  // collat -> debt vault -> [bid,ask]
	CollatLtvs           [][3]uint16       // collat -> debt vault
	IsOperatorAuthorized bool
}
