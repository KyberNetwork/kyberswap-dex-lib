package euler

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	eulerswap "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool

	status uint32 // 0 = unactivated, 1 = unlocked, 2 = locked

	fee, protocolFee                         *uint256.Int
	equilibriumReserve0, equilibriumReserve1 *uint256.Int
	reserve0, reserve1                       *uint256.Int
	priceX, priceY                           *uint256.Int
	concentrationX, concentrationY           *uint256.Int

	vault0, vault1       string
	eulerAccount         string
	protocolFeeRecipient string

	vaults []eulerswap.Vault
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra eulerswap.Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	p := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		vaults:              extra.Vaults,
		status:              extra.Pause,
		vault0:              staticExtra.Vault0,
		vault1:              staticExtra.Vault1,
		eulerAccount:        staticExtra.EulerAccount,
		fee:                 staticExtra.Fee,
		protocolFee:         staticExtra.ProtocolFee,
		equilibriumReserve0: staticExtra.EquilibriumReserve0,
		equilibriumReserve1: staticExtra.EquilibriumReserve1,
		reserve0:            bignumber.NewUint256(entityPool.Reserves[0]),
		reserve1:            bignumber.NewUint256(entityPool.Reserves[1]),
		priceX:              staticExtra.PriceX,
		priceY:              staticExtra.PriceY,
		concentrationX:      staticExtra.ConcentrationX,
		concentrationY:      staticExtra.ConcentrationY,
	}

	return p, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if s.status != 1 {
		return nil, ErrSwapIsPaused
	}

	_, amountOut, swapInfo, err := s.swap(true, indexIn == 0, amountIn)
	if err != nil {
		return nil, err
	}

	if amountOut.IsZero() {
		return nil, ErrInvalidAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:            DefaultGas,
		SwapInfo:       swapInfo,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	var (
		tokenAmountOut = param.TokenAmountOut
		tokenIn        = param.TokenIn
	)

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	if s.status != 1 {
		return nil, ErrSwapIsPaused
	}

	amountIn, _, swapInfo, err := s.swap(false, indexIn == 0, amountOut)
	if err != nil {
		return nil, err
	}

	if amountIn.IsZero() {
		return nil, ErrInvalidAmountIn
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:           DefaultGas,
		SwapInfo:      swapInfo,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		if swapInfo.NewReserve0 != nil {
			s.reserve0.Set(swapInfo.NewReserve0)
		}
		if swapInfo.NewReserve0 != nil {
			s.reserve1.Set(swapInfo.NewReserve1)
		}

		if swapInfo.ZeroForOne {
			s.vaults[0].Debt = new(uint256.Int).Sub(s.vaults[0].Debt, swapInfo.DebtRepaid)
		} else {
			s.vaults[1].Debt = new(uint256.Int).Sub(s.vaults[1].Debt, swapInfo.DebtRepaid)
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolExtra{
		Vault0:              s.vault0,
		Vault1:              s.vault1,
		EulerAccount:        s.eulerAccount,
		EquilibriumReserve0: s.equilibriumReserve0.ToBig(),
		EquilibriumReserve1: s.equilibriumReserve1.ToBig(),
		Fee:                 s.fee.ToBig(),
		ProtocolFee:         s.protocolFee.ToBig(),
		PriceY:              s.priceY.ToBig(),
		PriceX:              s.priceX.ToBig(),
		ConcentrationY:      s.concentrationY.ToBig(),
		ConcentrationX:      s.concentrationX.ToBig(),
		BlockNumber:         s.Info.BlockNumber,
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.reserve0 = p.reserve0.Clone()
	cloned.reserve1 = p.reserve1.Clone()
	cloned.vaults = lo.Map(p.vaults, func(item eulerswap.Vault, _ int) eulerswap.Vault {
		item.Debt = new(uint256.Int).Set(item.Debt)
		return item
	})
	return &cloned
}

func (s *PoolSimulator) swap(
	isExactIn,
	isZeroForOne bool,
	amountSpecified *uint256.Int,
) (*uint256.Int, *uint256.Int, SwapInfo, error) {
	var (
		amountIn, amountOut *uint256.Int
		err                 error
	)

	if isExactIn {
		amountIn = amountSpecified
		amountOut, err = s.computeQuote(amountIn, isExactIn, isZeroForOne)
		if err != nil {
			return nil, nil, SwapInfo{}, err
		}
	} else {
		amountOut = amountSpecified
		amountIn, err = s.computeQuote(amountOut, isExactIn, isZeroForOne)
		if err != nil {
			return nil, nil, SwapInfo{}, err
		}
	}

	vault := lo.Ternary(isZeroForOne, s.vaults[0], s.vaults[1])
	amountInWithoutFee, debtRepaid := depositAssets(amountIn, vault.Debt, s.fee, s.protocolFee, s.protocolFeeRecipient)

	var newReserve0, newReserve1 uint256.Int
	if isZeroForOne {
		newReserve0.Add(s.reserve0, amountInWithoutFee)
		newReserve1.Sub(s.reserve1, amountOut)
	} else {
		newReserve0.Sub(s.reserve0, amountOut)
		newReserve1.Add(s.reserve1, amountInWithoutFee)
	}

	if !s.verify(&newReserve0, &newReserve1) {
		return nil, nil, SwapInfo{}, ErrCurveViolation
	}

	return amountIn, amountOut, SwapInfo{
		NewReserve0: &newReserve0,
		NewReserve1: &newReserve1,
		DebtRepaid:  debtRepaid,
		ZeroForOne:  isZeroForOne,
	}, nil
}

func (s *PoolSimulator) computeQuote(amount *uint256.Int, isExactIn, isZeroForOne bool) (*uint256.Int, error) {
	var (
		amountWithFee, denominator uint256.Int
	)

	if isExactIn {
		amountWithFee.Mul(amount, s.fee)
		amountWithFee.Div(&amountWithFee, big256.BONE)
		amountWithFee.Sub(amount, &amountWithFee)
	}

	inLimit, outLimit, err := s.calcLimits(isZeroForOne)
	if err != nil {
		return nil, err
	}

	quote, err := s.findCurvePoint(&amountWithFee, isExactIn, isZeroForOne)
	if err != nil {
		return nil, err
	}

	if isExactIn {
		if amountWithFee.Gt(inLimit) || quote.Gt(outLimit) {
			return nil, ErrSwapLimitExceeded
		}
	} else {
		if amountWithFee.Gt(outLimit) || quote.Gt(inLimit) {
			return nil, ErrSwapLimitExceeded
		}
		quote.Mul(quote, big256.BONE)
		denominator.Sub(big256.BONE, s.fee)
		quote.Div(quote, &denominator)
	}

	return quote, nil
}

func (s *PoolSimulator) calcLimits(isZeroForOne bool) (*uint256.Int, *uint256.Int, error) {
	var inLimit, outLimit, maxDeposit, maxWithdraw uint256.Int

	inLimit.Set(maxUint112)
	outLimit.Set(maxUint112)

	vault := lo.Ternary(isZeroForOne, s.vaults[0], s.vaults[1])

	// Supply caps on input
	maxDeposit.Add(vault.Debt, vault.MaxDeposit)
	if maxDeposit.Lt(&inLimit) {
		inLimit.Set(&maxDeposit)
	}

	// Remaining reserves of output
	if isZeroForOne {
		if s.reserve1.Lt(&outLimit) {
			outLimit.Set(s.reserve1)
		}
	} else {
		if s.reserve0.Lt(&outLimit) {
			outLimit.Set(s.reserve0)
		}
	}

	// Remaining cash and borrow caps in output
	vault = lo.Ternary(isZeroForOne, s.vaults[1], s.vaults[0])
	if vault.Cash.Lt(&outLimit) {
		outLimit.Set(vault.Cash)
	}

	maxWithdraw.Set(vault.MaxWithdraw)
	if vault.TotalBorrows.Gt(&maxWithdraw) {
		maxWithdraw.SetUint64(0)
	} else {
		maxWithdraw.Sub(&maxWithdraw, vault.TotalBorrows)
	}

	if maxWithdraw.Lt(&outLimit) {
		maxWithdraw.Add(&maxWithdraw, vault.EulerAccountAssets)
		if maxWithdraw.Lt(&outLimit) {
			outLimit.Set(&maxWithdraw)
		}
	}

	return &inLimit, &outLimit, nil
}

func (s *PoolSimulator) verify(newReserve0, newReserve1 *uint256.Int) bool {
	if newReserve0.Gt(maxUint112) || newReserve1.Gt(maxUint112) {
		return false
	}

	var (
		yNew *uint256.Int
		err  error
	)

	if newReserve0.Cmp(s.equilibriumReserve0) >= 0 {
		if newReserve1.Cmp(s.equilibriumReserve1) >= 0 {
			return true
		}
		yNew, err = f(newReserve1, s.priceY, s.priceX, s.equilibriumReserve1, s.equilibriumReserve0, s.concentrationY)
		if err != nil {
			return false
		}

		return newReserve0.Cmp(yNew) >= 0
	} else {
		if newReserve1.Lt(s.equilibriumReserve1) {
			return false
		}
		yNew, err = f(newReserve0, s.priceX, s.priceY, s.equilibriumReserve0, s.equilibriumReserve1, s.concentrationX)
		if err != nil {
			return false
		}
		return newReserve1.Cmp(yNew) >= 0
	}
}

func (s *PoolSimulator) findCurvePoint(amount *uint256.Int, isExactIn bool, isZeroForOne bool) (*uint256.Int, error) {
	output := new(uint256.Int)

	if isExactIn {
		// exact in
		if isZeroForOne {
			// swap X in and Y out
			xNew := new(uint256.Int).Add(s.reserve0, amount)
			var yNew *uint256.Int
			var err error

			if xNew.Cmp(s.equilibriumReserve0) <= 0 {
				// remain on f()
				yNew, err = f(xNew, s.priceX, s.priceY, s.equilibriumReserve0, s.equilibriumReserve1, s.concentrationX)
				if err != nil {
					return nil, err
				}
			} else {
				// move to g()
				yNew, err = fInverse(xNew, s.priceY, s.priceX, s.equilibriumReserve1, s.equilibriumReserve0, s.concentrationY)
				if err != nil {
					return nil, err
				}
			}
			if s.reserve1.Gt(yNew) {
				output.Sub(s.reserve1, yNew)
				return output, nil
			}
			output.SetUint64(0)
			return output, nil
		} else {
			// swap Y in and X out
			yNew := new(uint256.Int).Add(s.reserve1, amount)
			var xNew *uint256.Int
			var err error

			if yNew.Cmp(s.equilibriumReserve1) <= 0 {
				// remain on g()
				xNew, err = f(yNew, s.priceY, s.priceX, s.equilibriumReserve1, s.equilibriumReserve0, s.concentrationY)
				if err != nil {
					return nil, err
				}
			} else {
				// move to f()
				xNew, err = fInverse(yNew, s.priceX, s.priceY, s.equilibriumReserve0, s.equilibriumReserve1, s.concentrationX)
				if err != nil {
					return nil, err
				}
			}
			if s.reserve0.Gt(xNew) {
				output.Sub(s.reserve0, xNew)
				return output, nil
			}
			output.SetUint64(0)
			return output, nil
		}
	} else {
		// exact out
		if isZeroForOne {
			// swap Y out and X in
			if !s.reserve1.Gt(amount) {
				return nil, ErrSwapLimitExceeded
			}
			yNew := new(uint256.Int).Sub(s.reserve1, amount)
			var xNew *uint256.Int
			var err error

			if yNew.Cmp(s.equilibriumReserve1) <= 0 {
				// remain on g()
				xNew, err = f(yNew, s.priceY, s.priceX, s.equilibriumReserve1, s.equilibriumReserve0, s.concentrationY)
				if err != nil {
					return nil, err
				}
			} else {
				// move to f()
				xNew, err = fInverse(yNew, s.priceX, s.priceY, s.equilibriumReserve0, s.equilibriumReserve1, s.concentrationX)
				if err != nil {
					return nil, err
				}
			}
			if xNew.Gt(s.reserve0) {
				output.Sub(xNew, s.reserve0)
				return output, nil
			}
			output.SetUint64(0)
			return output, nil
		} else {
			// swap X out and Y in
			if !s.reserve0.Gt(amount) {
				return nil, ErrSwapLimitExceeded
			}
			xNew := new(uint256.Int).Sub(s.reserve0, amount)
			var yNew *uint256.Int
			var err error

			if xNew.Cmp(s.equilibriumReserve0) <= 0 {
				// remain on f()
				yNew, err = f(xNew, s.priceX, s.priceY, s.equilibriumReserve0, s.equilibriumReserve1, s.concentrationX)
				if err != nil {
					return nil, err
				}
			} else {
				// move to g()
				yNew, err = fInverse(xNew, s.priceY, s.priceX, s.equilibriumReserve1, s.equilibriumReserve0, s.concentrationY)
				if err != nil {
					return nil, err
				}
			}
			if yNew.Gt(s.reserve1) {
				output.Sub(yNew, s.reserve1)
				return output, nil
			}
			output.SetUint64(0)
			return output, nil
		}
	}
}

func depositAssets(
	amount,
	vaultDebt,
	fee,
	protocolFee *uint256.Int,
	protocolFeeRecipient string,
) (*uint256.Int, *uint256.Int) {
	var (
		feeAmount, protocolFeeAmount, deposited uint256.Int
	)

	if amount.IsZero() {
		return uint256.NewInt(0), uint256.NewInt(0)
	}

	feeAmount.Mul(amount, fee)
	feeAmount.Div(&feeAmount, big256.BONE)

	if !strings.EqualFold(protocolFeeRecipient, valueobject.ZeroAddress) {
		protocolFeeAmount.Mul(&feeAmount, protocolFee)
		protocolFeeAmount.Div(&protocolFeeAmount, big256.BONE)

		if !protocolFeeAmount.IsZero() {
			amount.Sub(amount, &protocolFeeAmount)
			feeAmount.Sub(&feeAmount, &protocolFeeAmount)
		}
	}

	var repaid uint256.Int
	if amount.Gt(vaultDebt) {
		repaid.Set(vaultDebt)
	} else {
		repaid.Set(amount)
	}

	amount.Sub(amount, &repaid)
	deposited.Add(&deposited, &repaid)

	if amount.Sign() > 0 {
		deposited.Add(&deposited, amount)
	}

	if deposited.Gt(&feeAmount) {
		deposited.Sub(&deposited, &feeAmount)
		return &deposited, &repaid
	}

	return uint256.NewInt(0), &repaid
}
