package dexv2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StructsOverallTokenData struct {
	BorrowRate             *big.Int
	SupplyRate             *big.Int
	Fee                    *big.Int
	IsSupplyPaused         bool
	IsWithdrawPaused       bool
	IsBorrowPaused         bool
	IsPaybackPaused        bool
	IsPaused               bool
	LastStoredUtilization  *big.Int
	StorageUpdateThreshold *big.Int
	LastUpdateTimestamp    *big.Int
	SupplyExchangePrice    *big.Int
	BorrowExchangePrice    *big.Int
	SupplyRawInterest      *big.Int
	SupplyInterestFree     *big.Int
	BorrowRawInterest      *big.Int
	BorrowInterestFree     *big.Int
	TotalSupply            *big.Int
	TotalBorrow            *big.Int
	Revenue                *big.Int
	MaxUtilization         *big.Int
	RateData               StructsRateData
}

// StructsRateData is an auto generated low-level Go binding around an user-defined struct.
type StructsRateData struct {
	Version    *big.Int
	RateDataV1 StructsRateDataV1Params
	RateDataV2 StructsRateDataV2Params
}

// StructsRateDataV1Params is an auto generated low-level Go binding around an user-defined struct.
type StructsRateDataV1Params struct {
	Token                 common.Address
	Kink                  *big.Int
	RateAtUtilizationZero *big.Int
	RateAtUtilizationKink *big.Int
	RateAtUtilizationMax  *big.Int
}

// StructsRateDataV2Params is an auto generated low-level Go binding around an user-defined struct.
type StructsRateDataV2Params struct {
	Token                  common.Address
	Kink1                  *big.Int
	Kink2                  *big.Int
	RateAtUtilizationZero  *big.Int
	RateAtUtilizationKink1 *big.Int
	RateAtUtilizationKink2 *big.Int
	RateAtUtilizationMax   *big.Int
}

// StructsUserBorrowData is an auto generated low-level Go binding around an user-defined struct.
type StructsUserBorrowData struct {
	ModeWithInterest       bool
	IsBorrowPaused         bool
	IsPaybackPaused        bool
	IsPaused               bool
	Borrow                 *big.Int
	BorrowLimit            *big.Int
	LastUpdateTimestamp    *big.Int
	ExpandPercent          *big.Int
	ExpandDuration         *big.Int
	BaseBorrowLimit        *big.Int
	MaxBorrowLimit         *big.Int
	BorrowableUntilLimit   *big.Int
	Borrowable             *big.Int
	BorrowLimitUtilization *big.Int
}

// StructsUserSupplyData is an auto generated low-level Go binding around an user-defined struct.
type StructsUserSupplyData struct {
	ModeWithInterest       bool
	IsSupplyPaused         bool
	IsWithdrawPaused       bool
	IsPaused               bool
	Supply                 *big.Int
	WithdrawalLimit        *big.Int
	LastUpdateTimestamp    *big.Int
	ExpandPercent          *big.Int
	ExpandDuration         *big.Int
	BaseWithdrawalLimit    *big.Int
	WithdrawableUntilLimit *big.Int
	Withdrawable           *big.Int
	DecayEndTimestamp      *big.Int
	DecayAmount            *big.Int
}

// TokenBaseState is an auto generated low-level Go binding around an user-defined struct.
type TokenBaseState struct {
	FloatBalance                *big.Int
	MaxCap                      *big.Int
	UnaccountedBorrow           *big.Int
	DexSupplyAtLiquidity        StructsUserSupplyData
	DexBorrowAtLiquidity        StructsUserBorrowData
	LiquidityTokenData          StructsOverallTokenData
	AvailableForSwapOut         *big.Int
	AvailableForSwapOutWithdraw *big.Int
	AvailableForSwapOutBorrow   *big.Int
}
