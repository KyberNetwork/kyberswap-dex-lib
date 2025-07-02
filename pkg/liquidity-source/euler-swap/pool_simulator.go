package eulerswap

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type PoolSimulator struct {
	pool.Pool

	status uint32 // 0 = unactivated, 1 = unlocked, 2 = locked

	equilibriumReserve0, equilibriumReserve1 *uint256.Int
	reserve0, reserve1                       *uint256.Int

	priceX, priceY                 *uint256.Int
	concentrationX, concentrationY *uint256.Int

	fee, protocolFee     *uint256.Int
	protocolFeeRecipient common.Address

	vaults []Vault
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
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
		vaults:               extra.Vaults,
		status:               extra.Pause,
		fee:                  staticExtra.Fee,
		protocolFee:          staticExtra.ProtocolFee,
		equilibriumReserve0:  staticExtra.EquilibriumReserve0,
		equilibriumReserve1:  staticExtra.EquilibriumReserve1,
		reserve0:             bignumber.NewUint256(entityPool.Reserves[0]),
		reserve1:             bignumber.NewUint256(entityPool.Reserves[1]),
		priceX:               staticExtra.PriceX,
		priceY:               staticExtra.PriceY,
		concentrationX:       staticExtra.ConcentrationX,
		concentrationY:       staticExtra.ConcentrationY,
		protocolFeeRecipient: staticExtra.ProtocolFeeRecipient,
	}

	return p, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	if p.status != 1 {
		return nil, ErrSwapIsPaused
	}

	indexIn, indexOut := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	_, amountOut, swapInfo, err := p.swap(true, indexIn == 0, amountIn)
	if err != nil {
		return nil, err
	}

	if amountOut.IsZero() {
		return nil, ErrInvalidAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: p.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:            DefaultGas,
		SwapInfo:       swapInfo,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	var (
		tokenAmountOut = param.TokenAmountOut
		tokenIn        = param.TokenIn
	)

	if p.status != 1 {
		return nil, ErrSwapIsPaused
	}

	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	amountIn, _, swapInfo, err := p.swap(false, indexIn == 0, amountOut)
	if err != nil {
		return nil, err
	}

	if amountIn.IsZero() {
		return nil, ErrInvalidAmountIn
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: p.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: p.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:           DefaultGas,
		SwapInfo:      swapInfo,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		if swapInfo.NewReserve0 != nil {
			p.reserve0.Set(swapInfo.NewReserve0)
		}
		if swapInfo.NewReserve0 != nil {
			p.reserve1.Set(swapInfo.NewReserve1)
		}

		if swapInfo.ZeroForOne {
			p.vaults[0].Debt = new(uint256.Int).Sub(p.vaults[0].Debt, swapInfo.DebtRepaid)
		} else {
			p.vaults[1].Debt = new(uint256.Int).Sub(p.vaults[1].Debt, swapInfo.DebtRepaid)
		}
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	return PoolExtra{
		Fee:         p.fee,
		BlockNumber: p.Info.BlockNumber,
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.reserve0 = p.reserve0.Clone()
	cloned.reserve1 = p.reserve1.Clone()
	cloned.vaults = lo.Map(p.vaults, func(item Vault, _ int) Vault {
		item.Debt = new(uint256.Int).Set(item.Debt)
		return item
	})
	return &cloned
}

func (p *PoolSimulator) swap(
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
		amountOut, err = p.computeQuote(amountIn, isExactIn, isZeroForOne)
		if err != nil {
			return nil, nil, SwapInfo{}, err
		}
	} else {
		amountOut = amountSpecified
		amountIn, err = p.computeQuote(amountOut, isExactIn, isZeroForOne)
		if err != nil {
			return nil, nil, SwapInfo{}, err
		}
	}

	vault := lo.Ternary(isZeroForOne, p.vaults[0], p.vaults[1])
	amountInWithoutFee, debtRepaid := depositAssets(amountIn, vault.Debt, p.fee, p.protocolFee, p.protocolFeeRecipient)

	var newReserve0, newReserve1 uint256.Int
	if isZeroForOne {
		newReserve0.Add(p.reserve0, amountInWithoutFee)
		newReserve1.Sub(p.reserve1, amountOut)
	} else {
		newReserve0.Sub(p.reserve0, amountOut)
		newReserve1.Add(p.reserve1, amountInWithoutFee)
	}

	if !p.verify(&newReserve0, &newReserve1) {
		return nil, nil, SwapInfo{}, ErrCurveViolation
	}

	return amountIn, amountOut, SwapInfo{
		NewReserve0: &newReserve0,
		NewReserve1: &newReserve1,
		DebtRepaid:  debtRepaid,
		ZeroForOne:  isZeroForOne,
	}, nil
}

func (p *PoolSimulator) computeQuote(amount *uint256.Int, isExactIn, isZeroForOne bool) (*uint256.Int, error) {
	var (
		amountWithFee = new(uint256.Int).Set(amount)
		denominator   uint256.Int
	)

	if isExactIn {
		amountWithFee.Mul(amount, p.fee)
		amountWithFee.Div(amountWithFee, big256.BONE)
		amountWithFee.Sub(amount, amountWithFee)
	}

	inLimit, outLimit, err := p.calcLimits(isZeroForOne)
	if err != nil {
		return nil, err
	}

	quote, err := p.findCurvePoint(amountWithFee, isExactIn, isZeroForOne)
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
		denominator.Sub(big256.BONE, p.fee)
		quote.Div(quote, &denominator)
	}

	return quote, nil
}

func (p *PoolSimulator) calcLimits(isZeroForOne bool) (*uint256.Int, *uint256.Int, error) {
	var inLimit, outLimit, maxDeposit, maxWithdraw uint256.Int

	inLimit.Set(maxUint112)
	outLimit.Set(maxUint112)

	vault := lo.Ternary(isZeroForOne, p.vaults[0], p.vaults[1])

	// Supply caps on input
	maxDeposit.Add(vault.Debt, vault.MaxDeposit)
	if maxDeposit.Lt(&inLimit) {
		inLimit.Set(&maxDeposit)
	}

	// Remaining reserves of output
	if isZeroForOne {
		if p.reserve1.Lt(&outLimit) {
			outLimit.Set(p.reserve1)
		}
	} else {
		if p.reserve0.Lt(&outLimit) {
			outLimit.Set(p.reserve0)
		}
	}

	// Remaining cash and borrow caps in output
	vault = lo.Ternary(isZeroForOne, p.vaults[1], p.vaults[0])
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

func (p *PoolSimulator) verify(newReserve0, newReserve1 *uint256.Int) bool {
	if newReserve0.Gt(maxUint112) || newReserve1.Gt(maxUint112) {
		return false
	}

	var (
		yNew *uint256.Int
		err  error
	)

	if newReserve0.Cmp(p.equilibriumReserve0) >= 0 {
		if newReserve1.Cmp(p.equilibriumReserve1) >= 0 {
			return true
		}
		yNew, err = f(newReserve1, p.priceY, p.priceX, p.equilibriumReserve1, p.equilibriumReserve0, p.concentrationY)
		if err != nil {
			return false
		}

		return newReserve0.Cmp(yNew) >= 0
	} else {
		if newReserve1.Lt(p.equilibriumReserve1) {
			return false
		}
		yNew, err = f(newReserve0, p.priceX, p.priceY, p.equilibriumReserve0, p.equilibriumReserve1, p.concentrationX)
		if err != nil {
			return false
		}
		return newReserve1.Cmp(yNew) >= 0
	}
}

func (p *PoolSimulator) findCurvePoint(amount *uint256.Int, isExactIn bool, isZeroForOne bool) (*uint256.Int, error) {
	output := new(uint256.Int)

	if isExactIn {
		// exact in
		if isZeroForOne {
			// swap X in and Y out
			xNew := new(uint256.Int).Add(p.reserve0, amount)
			var yNew *uint256.Int
			var err error

			if xNew.Cmp(p.equilibriumReserve0) <= 0 {
				// remain on f()
				yNew, err = f(xNew, p.priceX, p.priceY, p.equilibriumReserve0, p.equilibriumReserve1, p.concentrationX)
				if err != nil {
					return nil, err
				}
			} else {
				// move to g()
				yNew, err = fInverse(xNew, p.priceY, p.priceX, p.equilibriumReserve1, p.equilibriumReserve0, p.concentrationY)
				if err != nil {
					return nil, err
				}
			}
			if p.reserve1.Gt(yNew) {
				output.Sub(p.reserve1, yNew)
				return output, nil
			}
			output.SetUint64(0)
			return output, nil
		} else {
			// swap Y in and X out
			yNew := new(uint256.Int).Add(p.reserve1, amount)
			var xNew *uint256.Int
			var err error

			if yNew.Cmp(p.equilibriumReserve1) <= 0 {
				// remain on g()
				xNew, err = f(yNew, p.priceY, p.priceX, p.equilibriumReserve1, p.equilibriumReserve0, p.concentrationY)
				if err != nil {
					return nil, err
				}
			} else {
				// move to f()
				xNew, err = fInverse(yNew, p.priceX, p.priceY, p.equilibriumReserve0, p.equilibriumReserve1, p.concentrationX)
				if err != nil {
					return nil, err
				}
			}
			if p.reserve0.Gt(xNew) {
				output.Sub(p.reserve0, xNew)
				return output, nil
			}
			output.SetUint64(0)
			return output, nil
		}
	} else {
		// exact out
		if isZeroForOne {
			// swap Y out and X in
			if !p.reserve1.Gt(amount) {
				return nil, ErrSwapLimitExceeded
			}
			yNew := new(uint256.Int).Sub(p.reserve1, amount)
			var xNew *uint256.Int
			var err error

			if yNew.Cmp(p.equilibriumReserve1) <= 0 {
				// remain on g()
				xNew, err = f(yNew, p.priceY, p.priceX, p.equilibriumReserve1, p.equilibriumReserve0, p.concentrationY)
				if err != nil {
					return nil, err
				}
			} else {
				// move to f()
				xNew, err = fInverse(yNew, p.priceX, p.priceY, p.equilibriumReserve0, p.equilibriumReserve1, p.concentrationX)
				if err != nil {
					return nil, err
				}
			}
			if xNew.Gt(p.reserve0) {
				output.Sub(xNew, p.reserve0)
				return output, nil
			}
			output.SetUint64(0)
			return output, nil
		} else {
			// swap X out and Y in
			if !p.reserve0.Gt(amount) {
				return nil, ErrSwapLimitExceeded
			}
			xNew := new(uint256.Int).Sub(p.reserve0, amount)
			var yNew *uint256.Int
			var err error

			if xNew.Cmp(p.equilibriumReserve0) <= 0 {
				// remain on f()
				yNew, err = f(xNew, p.priceX, p.priceY, p.equilibriumReserve0, p.equilibriumReserve1, p.concentrationX)
				if err != nil {
					return nil, err
				}
			} else {
				// move to g()
				yNew, err = fInverse(xNew, p.priceY, p.priceX, p.equilibriumReserve1, p.equilibriumReserve0, p.concentrationY)
				if err != nil {
					return nil, err
				}
			}
			if yNew.Gt(p.reserve1) {
				output.Sub(yNew, p.reserve1)
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
	protocolFeeRecipient common.Address,
) (*uint256.Int, *uint256.Int) {
	if amount.IsZero() {
		return uint256.NewInt(0), uint256.NewInt(0)
	}

	var remaining uint256.Int
	remaining.Set(amount)

	var feeAmount uint256.Int
	feeAmount.Mul(amount, fee)
	feeAmount.Div(&feeAmount, big256.BONE)

	if protocolFeeRecipient.Cmp(eth.AddressZero) != 0 {
		var protocolFeeAmount uint256.Int
		protocolFeeAmount.Mul(&feeAmount, protocolFee)
		protocolFeeAmount.Div(&protocolFeeAmount, big256.BONE)

		if !protocolFeeAmount.IsZero() {
			remaining.Sub(&remaining, &protocolFeeAmount)
			feeAmount.Sub(&feeAmount, &protocolFeeAmount)
		}
	}

	var repaid uint256.Int
	if remaining.Gt(vaultDebt) {
		repaid.Set(vaultDebt)
	} else {
		repaid.Set(&remaining)
	}

	remaining.Sub(&remaining, &repaid)

	var deposited uint256.Int
	deposited.Add(&deposited, &repaid)

	if remaining.Sign() > 0 {
		deposited.Add(&deposited, &remaining)
	}

	if deposited.Gt(&feeAmount) {
		deposited.Sub(&deposited, &feeAmount)
		return &deposited, &repaid
	}

	return uint256.NewInt(0), &repaid
}
