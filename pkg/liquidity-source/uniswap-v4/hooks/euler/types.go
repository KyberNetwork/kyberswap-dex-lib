package euler

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Vault0              string       `json:"v0"`
	Vault1              string       `json:"v1"`
	EulerAccount        string       `json:"ea"`
	Fee                 *uint256.Int `json:"f"`
	ProtocolFee         *uint256.Int `json:"pf"`
	EquilibriumReserve0 *uint256.Int `json:"er0"`
	EquilibriumReserve1 *uint256.Int `json:"er1"`
	PriceX              *uint256.Int `json:"px"`
	PriceY              *uint256.Int `json:"py"`
	ConcentrationX      *uint256.Int `json:"cx"`
	ConcentrationY      *uint256.Int `json:"cy"`
}

type SwapInfo struct {
	NewReserve0 *uint256.Int `json:"newReserve0"`
	NewReserve1 *uint256.Int `json:"newReserve1"`
	DebtRepaid  *uint256.Int `json:"debtRepaid"`
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

type PoolExtra struct {
	Vault0              string   `json:"vault0"`
	Vault1              string   `json:"vault1"`
	EulerAccount        string   `json:"eulerAccount"`
	EquilibriumReserve0 *big.Int `json:"equilibriumReserve0"`
	EquilibriumReserve1 *big.Int `json:"equilibriumReserve1"`
	Fee                 *big.Int `json:"fee"`
	ProtocolFee         *big.Int `json:"protocolFee"`
	PriceY              *big.Int `json:"priceY"`
	PriceX              *big.Int `json:"priceX"`
	ConcentrationY      *big.Int `json:"concentrationY"`
	ConcentrationX      *big.Int `json:"concentrationX"`
	BlockNumber         uint64   `json:"blockNumber"`
}
