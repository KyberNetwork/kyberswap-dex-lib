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
	Pause  uint32  `json:"p"`
	Vaults []Vault `json:"v"`
}

type Vault struct {
	Cash               *uint256.Int
	Debt               *uint256.Int
	MaxDeposit         *uint256.Int
	MaxWithdraw        *uint256.Int
	TotalBorrows       *uint256.Int
	EulerAccountAssets *uint256.Int
	CanBorrow          bool
}
type SwapInfo struct {
	NewReserve0 *uint256.Int
	NewReserve1 *uint256.Int
	DebtRepaid  *uint256.Int
	ZeroForOne  bool
}

type TrackerData struct {
	Vaults               []VaultRPC
	Reserves             ReserveRPC
	AccountLiquidities   []AccountLiquidityRPC
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
	EulerAccountBalance *big.Int
	TotalAssets         *big.Int
	TotalSupply         *big.Int
	Caps                [2]uint16
	MaxWithdraw         *big.Int
}

type AccountLiquidityRPC struct {
	CollateralValue *big.Int
	LiabilityValue  *big.Int
}
