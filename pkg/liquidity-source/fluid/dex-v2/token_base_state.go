package dexv2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
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

func liquidityLimitsFromRPC(b TokenBaseState, allowed [4]bool) (*LiquidityLimits, error) {
	s := &LiquidityLimits{Allowed: allowed}
	if b.DexSupplyAtLiquidity.WithdrawalLimit == nil {
		return nil, ErrUnsupportedLiquidityState
	}
	s.DepositSafe = b.DexSupplyAtLiquidity.WithdrawalLimit.Sign() > 0
	for _, entry := range []struct {
		src *big.Int
		dst *uint256.Int
	}{{b.FloatBalance, &s.Float}, {b.DexSupplyAtLiquidity.Withdrawable, &s.Withdraw}, {b.DexBorrowAtLiquidity.Borrowable, &s.Borrow}, {b.DexBorrowAtLiquidity.Borrow, &s.Debt}} {
		if entry.src == nil || entry.src.Sign() < 0 || entry.dst.SetFromBig(entry.src) {
			return nil, ErrUnsupportedLiquidityState
		}
	}
	s.Cash = s.Withdraw
	if s.Borrow.Gt(&s.Cash) {
		s.Cash = s.Borrow
	}
	var supply, borrow, price, other, precision, unit uint256.Int
	if b.LiquidityTokenData.TotalSupply == nil || b.LiquidityTokenData.TotalBorrow == nil || b.LiquidityTokenData.SupplyExchangePrice == nil || b.LiquidityTokenData.BorrowExchangePrice == nil {
		return nil, ErrUnsupportedLiquidityState
	}
	if supply.SetFromBig(b.LiquidityTokenData.TotalSupply) || borrow.SetFromBig(b.LiquidityTokenData.TotalBorrow) || price.SetFromBig(b.LiquidityTokenData.SupplyExchangePrice) || other.SetFromBig(b.LiquidityTokenData.BorrowExchangePrice) {
		return nil, ErrOverflow
	}
	precision.SetUint64(1000000000000)
	if price.Lt(&precision) || other.Lt(&precision) || price.BitLen() > 64 || other.BitLen() > 64 {
		return nil, ErrUnsupportedLiquidityState
	}
	if other.Gt(&price) {
		price = other
	}
	// Bound cumulative gross input and output per token to the current largest
	// aggregate. Thus neither aggregate can grow beyond twice this value, and
	// the rounding allowance also covers successive operations across pools.
	if borrow.Gt(&supply) {
		supply = borrow
	}
	if supply.IsZero() || supply.BitLen() > 125 {
		return nil, ErrUnsupportedLiquidityState
	}
	s.Growth = supply
	// Four times the initial aggregate leaves room for rounded raw positions;
	// 32 56-bit quanta cover per-operation user, limit and aggregate rounding,
	// plus exchange-price conversions. Deposits are debited separately, so
	// no benefit from their potentially rounded-down decay amount is assumed.
	shift := max(supply.BitLen()+2-56, 0)
	unit.Add(&price, &precision).SubUint64(&unit, 1).Div(&unit, &precision)
	s.RoundingBuffer.SetOne().Lsh(&s.RoundingBuffer, uint(shift))
	if _, overflow := s.RoundingBuffer.MulOverflow(&s.RoundingBuffer, &unit); overflow {
		return nil, ErrOverflow
	}
	factor := uint256.Int{}
	factor.SetUint64(32)
	if _, overflow := s.RoundingBuffer.MulOverflow(&s.RoundingBuffer, &factor); overflow {
		return nil, ErrOverflow
	}
	if !s.DepositSafe {
		var base, current uint256.Int
		if b.DexSupplyAtLiquidity.BaseWithdrawalLimit == nil || b.DexSupplyAtLiquidity.Supply == nil || base.SetFromBig(b.DexSupplyAtLiquidity.BaseWithdrawalLimit) || current.SetFromBig(b.DexSupplyAtLiquidity.Supply) {
			return nil, ErrUnsupportedLiquidityState
		}
		if base.Gt(&current) {
			s.DepositHeadroom.Sub(&base, &current)
			if s.DepositHeadroom.Gt(&s.RoundingBuffer) {
				s.DepositHeadroom.Sub(&s.DepositHeadroom, &s.RoundingBuffer)
			} else {
				s.DepositHeadroom.Clear()
			}
		}
	}
	return s, nil
}

// The deployed Ethereum user module uses bit 255 as a blanket pause. The
// newer resolver's per-operation pause flags alone do not capture that rule.
func liquidityPermissions(b TokenBaseState, supply, borrow, config, status *big.Int, timestamp uint64) ([4]bool, error) {
	if supply == nil || borrow == nil || config == nil || status == nil {
		return [4]bool{}, ErrUnsupportedLiquidityState
	}
	tokenAllowed := config.Bit(255) == 0 && status.Cmp(big.NewInt(2)) != 0
	supplyAllowed := supply.Sign() != 0 && supply.Bit(255) == 0
	borrowAllowed := borrow.Sign() != 0 && borrow.Bit(255) == 0

	s, r, t := b.DexSupplyAtLiquidity, b.DexBorrowAtLiquidity, b.LiquidityTokenData
	return [4]bool{
		tokenAllowed && supplyAllowed && !s.IsPaused && !s.IsSupplyPaused && !t.IsPaused && !t.IsSupplyPaused,
		tokenAllowed && supplyAllowed && !s.IsPaused && !s.IsWithdrawPaused && !t.IsPaused && !t.IsWithdrawPaused,
		tokenAllowed && borrowAllowed && !r.IsPaused && !r.IsBorrowPaused && !t.IsPaused && !t.IsBorrowPaused,
		tokenAllowed && borrowAllowed && !r.IsPaused && !r.IsPaybackPaused && !t.IsPaused && !t.IsPaybackPaused,
	}, nil
}
