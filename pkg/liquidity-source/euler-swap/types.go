package eulerswap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type PoolExtra struct {
	Fee         *uint256.Int `json:"fee"`
	BlockNumber uint64       `json:"blockNumber"`
}

type StaticExtra struct {
	Vault0               string         `json:"v0"`
	Vault1               string         `json:"v1"`
	EulerAccount         string         `json:"ea"`
	Fee                  *uint256.Int   `json:"f"`
	ProtocolFee          *uint256.Int   `json:"pf"`
	EquilibriumReserve0  *uint256.Int   `json:"er0"`
	EquilibriumReserve1  *uint256.Int   `json:"er1"`
	PriceX               *uint256.Int   `json:"px"`
	PriceY               *uint256.Int   `json:"py"`
	ConcentrationX       *uint256.Int   `json:"cx"`
	ConcentrationY       *uint256.Int   `json:"cy"`
	ProtocolFeeRecipient common.Address `json:"pfr"`
	EVC                  string         `json:"evc"`
}

type Extra struct {
	Pause           uint32         `json:"p,omitempty"`  // 0 = unactivated, 1 = unlocked, 2 = locked
	Vaults          [3]*Vault      `json:"v"`            // vault0, vault1, controllerVault
	ControllerVault string         `json:"cV,omitempty"` // controller vault address
	Collaterals     []*uint256.Int `json:"c,omitempty"`  // collateral amounts across all collateral vaults
}

type VaultInfo struct {
	VaultAddress string
	AssetAddress string
	QuoteAmount  *big.Int
}

type Vault struct {
	Cash                *uint256.Int    `json:"c,omitempty"` // ~ totalAssets - totalBorrows
	Debt                *uint256.Int    `json:"d,omitempty"` // debt of euler account
	MaxDeposit          *uint256.Int    `json:"mD,omitempty"`
	MaxWithdraw         *uint256.Int    `json:"mW,omitempty"`
	TotalBorrows        *uint256.Int    `json:"tB,omitempty"`
	EulerAccountAssets  *uint256.Int    `json:"eAA,omitempty"`
	DebtPrice           *uint256.Int    `json:"dP,omitempty"`   // quoted debt price against itself
	ValuePrices         []*uint256.Int  `json:"vP,omitempty"`   // quoted value prices against collaterals
	VaultValuePrices    [2]*uint256.Int `json:"vVP,omitempty"`  // quoted value prices against v0 + v1
	LTVs                []uint64        `json:"ltv,omitempty"`  // borrow ltv against each collateral
	VaultLTVs           [2]uint64       `json:"vLtv,omitempty"` // borrow ltv against v0 + v1
	IsControllerEnabled bool            `json:"iCE,omitempty"`  // is controller enabled
}

type SwapInfo struct {
	reserves              [2]*uint256.Int
	withdrawAmount        *uint256.Int // withdrawn collateral asset amount
	borrowAmount          *uint256.Int // new buy token debt amount
	depositAmount         *uint256.Int // amount in after fee
	repayAmount           *uint256.Int // part of amount in after fee used to repay
	debt                  *uint256.Int
	debtVaultIdx          int
	collateralValue       *uint256.Int
	isSellVaultControlled bool
	isBuyVaultControlled  bool
	ZeroForOne            bool `json:"zeroForOne"`
}

type TrackerData struct {
	Vaults               []VaultRPC
	Reserves             ReserveRPC
	Controller           string            // controller debt vault, if exist
	VaultPrices          [3][3][2]*big.Int // other vault -> debt vault -> [bid/value,ask/debt]
	VaultLtvs            [3][3]uint16      // vault 0/1/controller -> debt vault
	CollatAmts           []*big.Int        // asset amount of euler account across collateral vaults
	CollatPrices         [][3][2]*big.Int  // collat -> debt vault -> [bid,ask]
	CollatLtvs           [][3]uint16       // collat -> debt vault
	IsOperatorAuthorized bool
}

type ReserveRPC struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
	Status   uint32
}

type ParamsRPC struct {
	Data struct {
		Vault0               common.Address `abi:"vault0"`
		Vault1               common.Address `abi:"vault1"`
		EulerAccount         common.Address `abi:"eulerAccount"`
		EquilibriumReserve0  *big.Int       `abi:"equilibriumReserve0"`
		EquilibriumReserve1  *big.Int       `abi:"equilibriumReserve1"`
		PriceX               *big.Int       `abi:"priceX"`
		PriceY               *big.Int       `abi:"priceY"`
		ConcentrationX       *big.Int       `abi:"concentrationX"`
		ConcentrationY       *big.Int       `abi:"concentrationY"`
		Fee                  *big.Int       `abi:"fee"`
		ProtocolFee          *big.Int       `abi:"protocolFee"`
		ProtocolFeeRecipient common.Address `abi:"protocolFeeRecipient"`
	}
}

type VaultRPC struct {
	Cash                *big.Int
	Debt                *big.Int
	MaxDeposit          *big.Int
	TotalBorrows        *big.Int
	TotalAssets         *big.Int
	TotalSupply         *big.Int
	EulerAccountBalance *big.Int
	MaxWithdraw         *big.Int
	Caps                [2]uint16
	IsControllerEnabled bool
}

type AccountLiquidityRPC struct {
	CollateralValue *big.Int
	LiabilityValue  *big.Int
}
