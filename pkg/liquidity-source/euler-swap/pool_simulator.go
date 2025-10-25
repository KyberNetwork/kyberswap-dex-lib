package eulerswap

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	StaticExtra
	Extra
	reserves        [2]*uint256.Int
	collateralValue *uint256.Int
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

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		collateralValue: uint256.NewInt(0),
		reserves:        [2]*uint256.Int{big256.New(entityPool.Reserves[0]), big256.New(entityPool.Reserves[1])},
		StaticExtra:     staticExtra,
		Extra:           extra,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.Pause != 1 {
		return nil, ErrSwapIsPaused
	}

	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
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

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: p.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:            DefaultGas,
		SwapInfo:       swapInfo,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if p.Pause != 1 {
		return nil, ErrSwapIsPaused
	}

	tokenAmountOut, tokenIn := param.TokenAmountOut, param.TokenIn
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

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: p.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: p.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:           DefaultGas,
		SwapInfo:      swapInfo,
	}, nil
}

func (p *PoolSimulator) swap(
	isExactIn,
	zeroForOne bool,
	amountSpecified *uint256.Int,
) (amountIn, amountOut *uint256.Int, swapInfo *SwapInfo, err error) {
	if isExactIn {
		amountIn = amountSpecified
		amountOut, err = p.computeQuote(amountIn, isExactIn, zeroForOne)
		if err != nil {
			return nil, nil, swapInfo, err
		}
	} else {
		amountOut = amountSpecified
		amountIn, err = p.computeQuote(amountOut, isExactIn, zeroForOne)
		if err != nil {
			return nil, nil, swapInfo, err
		}
	}

	swapInfo, err = p.updateAndCheckSolvency(amountIn, amountOut, zeroForOne)
	return amountIn, amountOut, swapInfo, err
}

// https://www.notion.so/kybernetwork/EulerSwap-updateAndCheckSolvency-27426751887e807c915ac66c95512a4a
func (p *PoolSimulator) updateAndCheckSolvency(
	amtIn,
	amtOut *uint256.Int,
	zeroForOne bool,
) (*SwapInfo, error) {
	debtVaultAddr, debtVaultIdx, debt := p.ControllerVault, 2, uint256.NewInt(0)
	if p.Vaults[2] != nil {
		debt = p.Vaults[2].Debt.Clone()
	}
	sellVaultAddr, buyVaultAddr, sellVaultIdx, buyVaultIdx := p.Vault1, p.Vault0, 1, 0
	if zeroForOne { // user sells tokenIn from sellVault to get tokenOut from buyVault
		sellVaultAddr, buyVaultAddr, sellVaultIdx, buyVaultIdx = p.Vault0, p.Vault1, 0, 1
	}
	sellVault, buyVault := p.Vaults[sellVaultIdx], p.Vaults[buyVaultIdx]

	// soldCollat = buy token (tokenOut) given to user; newDebt = new debt in buy token
	soldCollat, newDebt, isBuyVaultControlled := withdrawAssets(
		amtOut, buyVault.EulerAccountAssets, buyVault.IsControllerEnabled)

	depositAmt, repayAmt, feeAmt, isSellVaultControlled := depositAssets(
		amtIn, p.Fee, sellVault.Debt, p.ProtocolFee, p.ProtocolFeeRecipient, sellVault.IsControllerEnabled)

	newCollat := uint256.NewInt(0) // new sell token collateral (tokenIn) after swap
	if strings.EqualFold(debtVaultAddr, sellVaultAddr) {
		if depositAmt.Lt(debt) { // partial repayment of controller vault
			if newDebt.Sign() > 0 { // left-over debt in controller vault + new debt in buy vault = forbidden
				return nil, ErrMultiDebts
			}
			debt.Sub(debt, depositAmt)
		} else {
			newCollat.Sub(depositAmt, debt)
			debt.Clear()
		}
	} else {
		newCollat = depositAmt
	}

	if newDebt.Sign() > 0 {
		if !strings.EqualFold(debtVaultAddr, buyVaultAddr) {
			if debt.Sign() > 0 { // unpaid debt in controller vault + new debt in buy vault = forbidden
				return nil, ErrMultiDebts
			}
			debtVaultIdx = buyVaultIdx
		}
		debt.Add(debt, newDebt)
	}

	collatVal := p.collateralValue.Clone()
	if debt.Sign() > 0 {
		debtVault := p.Vaults[debtVaultIdx]
		valuePrices, ltvs := debtVault.ValuePrices, debtVault.LTVs

		var liabilityVal, tmp uint256.Int
		liabilityVal.Mul(debt, debtVault.DebtPrice).Mul(&liabilityVal, big256.UBasisPoint)

		// the sum of all LTV-adjusted, unit-of-account valued collaterals
		if collatVal.IsZero() {
			for i, collateral := range p.Collaterals {
				collatVal.Add(collatVal, tmp.Mul(tmp.Mul(collateral, tmp.SetUint64(ltvs[i])), valuePrices[i]))
			}
		}

		vaultValuePrices, vaultLtvs := debtVault.VaultValuePrices, debtVault.VaultLTVs
		collatVal.Add(collatVal,
			tmp.Mul(tmp.Mul(newCollat, tmp.SetUint64(vaultLtvs[sellVaultIdx])), vaultValuePrices[sellVaultIdx]))

		collatVal.Sub(collatVal,
			tmp.Mul(tmp.Mul(soldCollat, tmp.SetUint64(vaultLtvs[buyVaultIdx])), vaultValuePrices[buyVaultIdx]))

		// Apply a safety buffer (85%) to the collateral value for swap limit checks
		collatValWithBuffer, _ := tmp.MulDivOverflow(collatVal, bufferSwapLimit, big256.UBasisPoint)
		if liabilityVal.Gt(collatValWithBuffer) {
			return nil, ErrInsolvency
		}
	}

	var newReserve0, newReserve1 uint256.Int
	if zeroForOne {
		if depositAmt.Gt(feeAmt) {
			newReserve0.Add(p.reserves[0], depositAmt)
			newReserve0.Sub(&newReserve0, feeAmt)
		}
		newReserve1.Sub(p.reserves[1], amtOut)
	} else {
		if depositAmt.Gt(feeAmt) {
			newReserve1.Add(p.reserves[1], depositAmt)
			newReserve1.Sub(&newReserve1, feeAmt)
		}
		newReserve0.Sub(p.reserves[0], amtOut)
	}

	if !p.verify(&newReserve0, &newReserve1) {
		return nil, ErrCurveViolation
	}

	return &SwapInfo{
		reserves:              [2]*uint256.Int{&newReserve0, &newReserve1},
		withdrawAmount:        soldCollat,
		borrowAmount:          newDebt,
		depositAmount:         depositAmt,
		repayAmount:           repayAmt,
		debt:                  debt,
		debtVaultIdx:          debtVaultIdx,
		collateralValue:       collatVal,
		isSellVaultControlled: isSellVaultControlled,
		isBuyVaultControlled:  isBuyVaultControlled,
		ZeroForOne:            zeroForOne,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.reserves = [2]*uint256.Int(lo.Map(p.reserves[:], func(r *uint256.Int, _ int) *uint256.Int {
		return r.Clone()
	}))

	cloned.collateralValue = p.collateralValue.Clone()

	cloned.Vaults = [3]*Vault(lo.Map(p.Vaults[:], func(v *Vault, _ int) *Vault {
		if v == nil {
			return nil
		}
		clonedVault := *v
		return &clonedVault
	}))
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(*SwapInfo)
	if !ok {
		return
	}
	p.reserves = swapInfo.reserves
	from, to := 0, 1
	if !swapInfo.ZeroForOne {
		from, to = to, from
	}
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)

	depositAmt, repayAmt := swapInfo.depositAmount, swapInfo.repayAmount
	sellVault := p.Vaults[from]
	sellVault.IsControllerEnabled = swapInfo.isSellVaultControlled
	sellVault.Cash = new(uint256.Int).Add(sellVault.Cash, depositAmt)
	sellVault.Debt = subTill0(sellVault.Debt, repayAmt)
	sellVault.MaxDeposit = subTill0(sellVault.MaxDeposit, depositAmt)
	sellVault.TotalBorrows = subTill0(sellVault.TotalBorrows, repayAmt)
	addedAssets := subTill0(depositAmt, repayAmt)
	sellVault.EulerAccountAssets = addedAssets.Add(sellVault.EulerAccountAssets, addedAssets)

	withdrawAmt, borrowAmt := swapInfo.withdrawAmount, swapInfo.borrowAmount
	buyVault := p.Vaults[to]
	buyVault.IsControllerEnabled = swapInfo.isBuyVaultControlled
	buyVault.Cash = subTill0(buyVault.Cash, amountOut)
	buyVault.Debt = new(uint256.Int).Add(buyVault.Debt, borrowAmt)
	buyVault.MaxWithdraw = subTill0(buyVault.MaxWithdraw, withdrawAmt)
	buyVault.TotalBorrows = new(uint256.Int).Add(buyVault.TotalBorrows, borrowAmt)
	buyVault.EulerAccountAssets = subTill0(buyVault.EulerAccountAssets, withdrawAmt)

	if swapInfo.debtVaultIdx < 2 {
		p.Vaults[2] = p.Vaults[swapInfo.debtVaultIdx]
		p.ControllerVault = lo.Ternary(swapInfo.debtVaultIdx == 0, p.Vault0, p.Vault1)
	}

	if p.Vaults[2] != nil {
		p.Vaults[2].Debt = swapInfo.debt
	}

	p.collateralValue = swapInfo.collateralValue
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolExtra{
		Fee:         p.Fee,
		BlockNumber: p.Info.BlockNumber,
	}
}

func (p *PoolSimulator) computeQuote(amount *uint256.Int, isExactIn, isZeroForOne bool) (*uint256.Int, error) {
	var (
		amountWithFee = new(uint256.Int).Set(amount)
		denominator   uint256.Int
	)

	if isExactIn {
		amountWithFee.MulDivOverflow(amount, p.Fee, big256.BONE)
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
		denominator.Sub(big256.BONE, p.Fee)
		quote.Div(quote, &denominator)
	}

	return quote, nil
}

func (p *PoolSimulator) calcLimits(isZeroForOne bool) (*uint256.Int, *uint256.Int, error) {
	var inLimit, outLimit, maxDeposit, maxWithdraw uint256.Int

	inLimit.Set(maxUint112)
	outLimit.Set(maxUint112)

	vault := lo.Ternary(isZeroForOne, p.Vaults[0], p.Vaults[1])

	// Supply caps on input
	maxDeposit.Add(vault.Debt, vault.MaxDeposit)
	if maxDeposit.Lt(&inLimit) {
		inLimit.Set(&maxDeposit)
	}

	// Remaining reserves of output
	if isZeroForOne {
		if p.reserves[1].Lt(&outLimit) {
			outLimit.Set(p.reserves[1])
		}
	} else {
		if p.reserves[0].Lt(&outLimit) {
			outLimit.Set(p.reserves[0])
		}
	}

	// Remaining cash and borrow caps in output
	vault = lo.Ternary(isZeroForOne, p.Vaults[1], p.Vaults[0])
	if vault.Cash.Lt(&outLimit) {
		outLimit.Set(vault.Cash)
	}

	maxWithdraw.Set(vault.MaxWithdraw)
	if vault.TotalBorrows.Gt(&maxWithdraw) {
		maxWithdraw.Clear()
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

	if newReserve0.Cmp(p.EquilibriumReserve0) >= 0 {
		if newReserve1.Cmp(p.EquilibriumReserve1) >= 0 {
			return true
		}
		yNew, err = f(newReserve1, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0, p.ConcentrationY)
		if err != nil {
			return false
		}

		return newReserve0.Cmp(yNew) >= 0
	}

	if newReserve1.Lt(p.EquilibriumReserve1) {
		return false
	}
	yNew, err = f(newReserve0, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1, p.ConcentrationX)
	if err != nil {
		return false
	}
	return newReserve1.Cmp(yNew) >= 0
}

func (p *PoolSimulator) findCurvePoint(amount *uint256.Int, isExactIn bool, isZeroForOne bool) (*uint256.Int, error) {
	output := new(uint256.Int)

	if isExactIn {
		// exact in
		if isZeroForOne {
			// swap X in and Y out
			xNew := new(uint256.Int).Add(p.reserves[0], amount)
			var yNew *uint256.Int
			var err error

			if xNew.Cmp(p.EquilibriumReserve0) <= 0 {
				// remain on f()
				yNew, err = f(xNew, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1, p.ConcentrationX)
				if err != nil {
					return nil, err
				}
			} else {
				// move to g()
				yNew, err = fInverse(xNew, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0,
					p.ConcentrationY)
				if err != nil {
					return nil, err
				}
			}
			if p.reserves[1].Gt(yNew) {
				output.Sub(p.reserves[1], yNew)
				return output, nil
			}
			output.Clear()
			return output, nil
		}

		// swap Y in and X out
		yNew := new(uint256.Int).Add(p.reserves[1], amount)
		var xNew *uint256.Int
		var err error

		if yNew.Cmp(p.EquilibriumReserve1) <= 0 {
			// remain on g()
			xNew, err = f(yNew, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0, p.ConcentrationY)
			if err != nil {
				return nil, err
			}
		} else {
			// move to f()
			xNew, err = fInverse(yNew, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1,
				p.ConcentrationX)
			if err != nil {
				return nil, err
			}
		}
		if p.reserves[0].Gt(xNew) {
			output.Sub(p.reserves[0], xNew)
			return output, nil
		}
		output.Clear()
		return output, nil
	}

	// exact out
	if isZeroForOne {
		// swap Y out and X in
		if !p.reserves[1].Gt(amount) {
			return nil, ErrSwapLimitExceeded
		}
		yNew := new(uint256.Int).Sub(p.reserves[1], amount)
		var xNew *uint256.Int
		var err error

		if yNew.Cmp(p.EquilibriumReserve1) <= 0 {
			// remain on g()
			xNew, err = f(yNew, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0, p.ConcentrationY)
			if err != nil {
				return nil, err
			}
		} else {
			// move to f()
			xNew, err = fInverse(yNew, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1,
				p.ConcentrationX)
			if err != nil {
				return nil, err
			}
		}
		if xNew.Gt(p.reserves[0]) {
			output.Sub(xNew, p.reserves[0])
			return output, nil
		}
		output.Clear()
		return output, nil
	}

	// swap X out and Y in
	if !p.reserves[0].Gt(amount) {
		return nil, ErrSwapLimitExceeded
	}
	xNew := new(uint256.Int).Sub(p.reserves[0], amount)
	var yNew *uint256.Int
	var err error

	if xNew.Cmp(p.EquilibriumReserve0) <= 0 {
		// remain on f()
		yNew, err = f(xNew, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1, p.ConcentrationX)
		if err != nil {
			return nil, err
		}
	} else {
		// move to g()
		yNew, err = fInverse(xNew, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0,
			p.ConcentrationY)
		if err != nil {
			return nil, err
		}
	}
	if yNew.Gt(p.reserves[1]) {
		output.Sub(yNew, p.reserves[1])
		return output, nil
	}
	output.Clear()
	return output, nil
}

func withdrawAssets(amount, balance *uint256.Int, isControllerEnabled bool) (soldCollat, newDebt *uint256.Int, _ bool) {
	if amount.Cmp(balance) <= 0 {
		return amount, big256.U0, isControllerEnabled
	}

	return balance, new(uint256.Int).Sub(amount, balance), true
}

func depositAssets(
	amount,
	fee,
	debt,
	protocolFee *uint256.Int,
	protocolFeeRecipient common.Address,
	isControllerEnabled bool,
) (deposited, repaid, feeAmount *uint256.Int, _ bool) {
	if amount.IsZero() {
		return big256.U0, big256.U0, big256.U0, isControllerEnabled
	}

	feeAmount, _ = new(uint256.Int).MulDivOverflow(amount, fee, big256.BONE)

	remainingAmount := amount.Clone()
	if !valueobject.IsZeroAddress(protocolFeeRecipient) {
		var protocolFeeAmount uint256.Int
		if protocolFeeAmount.MulDivOverflow(feeAmount, protocolFee, big256.BONE); protocolFeeAmount.Sign() > 0 {
			remainingAmount.Sub(remainingAmount, &protocolFeeAmount)
			feeAmount.Sub(feeAmount, &protocolFeeAmount)
		}
	}

	repaid = uint256.NewInt(0)
	deposited = uint256.NewInt(0)

	if isControllerEnabled {
		if remainingAmount.Gt(debt) {
			repaid.Set(debt)
		} else {
			repaid.Set(remainingAmount)
		}

		remainingAmount.Sub(remainingAmount, repaid)
		deposited.Add(deposited, repaid)

		if debt.Eq(repaid) {
			isControllerEnabled = false // repaid 100%
		}
	}

	// Add remaining amount to deposited
	deposited.Add(deposited, remainingAmount)

	return deposited, repaid, feeAmount, isControllerEnabled
}

func subTill0(amt, sub *uint256.Int) *uint256.Int {
	if sub.Sign() == 0 {
		return amt
	}
	if sub.Cmp(amt) >= 0 {
		return big256.U0
	}
	return new(uint256.Int).Sub(amt, sub)
}
