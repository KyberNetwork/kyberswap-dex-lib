package v2

import (
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/v2/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	StaticExtra
	DynamicParams
	Pause           uint32
	SupplyVault     [2]*shared.VaultState
	BorrowVault     [3]*shared.VaultState
	ControllerVault string
	Collaterals     []*uint256.Int
	reserves        [2]*uint256.Int
	collateralValue *uint256.Int
	hook            hooks.Hook
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

	var hook hooks.Hook
	if extra.SwapHookedOps != 0 && extra.SwapHook != "" {
		hookAddr := common.HexToAddress(extra.SwapHook)
		hook = hooks.GetHook(hookAddr, &hooks.HookParam{
			Pool:        &entityPool,
			HookAddress: hookAddr,
			HookExtra:   extra.HookExtra,
		})
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
		StaticExtra:     staticExtra,
		DynamicParams:   extra.DynamicParams,
		Pause:           extra.Pause,
		SupplyVault:     extra.SupplyVault,
		BorrowVault:     extra.BorrowVault,
		ControllerVault: extra.ControllerVault,
		Collaterals:     extra.Collaterals,
		reserves:        [2]*uint256.Int{big256.New(entityPool.Reserves[0]), big256.New(entityPool.Reserves[1])},
		collateralValue: uint256.NewInt(0),
		hook:            hook,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if err := p.checkSwappable(); err != nil {
		return nil, err
	}

	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	indexIn, indexOut := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountIn
	}

	zeroForOne := indexIn == 0
	_, amountOut, swapInfo, err := p.swap(true, zeroForOne, amountIn)
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
	if err := p.checkSwappable(); err != nil {
		return nil, err
	}

	tokenAmountOut, tokenIn := param.TokenAmountOut, param.TokenIn
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountOut
	}

	zeroForOne := indexIn == 0
	amountIn, _, swapInfo, err := p.swap(false, zeroForOne, amountOut)
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

func (p *PoolSimulator) checkSwappable() error {
	if p.Pause != 1 {
		return shared.ErrSwapIsPaused
	}

	if p.Expiration != 0 && p.Expiration <= uint64(time.Now().Unix()) {
		return shared.ErrSwapExpired
	}

	return nil
}

func (p *PoolSimulator) swap(
	isAmountIn,
	zeroForOne bool,
	amount *uint256.Int,
) (amtIn, amtOut *uint256.Int, swapInfo *shared.SwapInfo, err error) {
	if p.SwapHookedOps&hooks.HookBeforeSwap != 0 && p.hook != nil {
		if err := p.hook.BeforeSwap(&hooks.BeforeSwapParams{
			AmountOut:  amount,
			ZeroForOne: zeroForOne,
		}); err != nil {
			return nil, nil, nil, err
		}
	}

	if isAmountIn {
		amtIn = amount
		amtOut, err = p.computeQuote(amtIn, isAmountIn, zeroForOne)
		if err != nil {
			return nil, nil, swapInfo, err
		}
	} else {
		amtOut = amount
		amtIn, err = p.computeQuote(amtOut, isAmountIn, zeroForOne)
		if err != nil {
			return nil, nil, swapInfo, err
		}
	}

	swapInfo, err = p.updateAndCheckSolvency(amtIn, amtOut, zeroForOne)
	if err != nil {
		return amtIn, amtOut, swapInfo, err
	}

	if p.SwapHookedOps&hooks.HookAfterSwap != 0 && p.hook != nil {
		fee := p.getFee(zeroForOne)
		feeAmt := new(uint256.Int).Mul(amtIn, fee)
		feeAmt.Div(feeAmt, big256.BONE)

		if err := p.hook.AfterSwap(&hooks.AfterSwapParams{
			AmountIn:   amtIn,
			AmountOut:  amtOut,
			Fee:        feeAmt,
			ZeroForOne: zeroForOne,
			Reserve0:   swapInfo.Reserves[0],
			Reserve1:   swapInfo.Reserves[1],
		}); err != nil {
			return nil, nil, nil, err
		}
	}

	return amtIn, amtOut, swapInfo, err
}

func (p *PoolSimulator) updateAndCheckSolvency(
	amtIn,
	amtOut *uint256.Int,
	zeroForOne bool,
) (*shared.SwapInfo, error) {
	debtVaultAddr, debtVaultIdx, debt := p.ControllerVault, 2, uint256.NewInt(0)

	controllerVault := p.BorrowVault[2]
	if p.BorrowVault[0] != nil && p.BorrowVault[0].IsControllerEnabled {
		controllerVault = p.BorrowVault[0]
		debtVaultAddr = p.BorrowVault0
		debtVaultIdx = 0
	} else if p.BorrowVault[1] != nil && p.BorrowVault[1].IsControllerEnabled {
		controllerVault = p.BorrowVault[1]
		debtVaultAddr = p.BorrowVault1
		debtVaultIdx = 1
	}

	if controllerVault != nil && controllerVault.Debt != nil {
		debt = controllerVault.Debt.Clone()
	}

	sellVaultIdx, buyVaultIdx := 1, 0
	if zeroForOne {
		sellVaultIdx, buyVaultIdx = 0, 1
	}
	sellBorrowVault := p.BorrowVault[sellVaultIdx]
	sellSupplyVault, buySupplyVault := p.SupplyVault[sellVaultIdx], p.SupplyVault[buyVaultIdx]
	sellBorrowVaultAddr := lo.Ternary(zeroForOne, p.BorrowVault0, p.BorrowVault1)
	buyBorrowVaultAddr := lo.Ternary(zeroForOne, p.BorrowVault1, p.BorrowVault0)

	// Withdraw: first from supply vault, then borrow from borrow vault
	var soldCollat, newDebt *uint256.Int
	var isBuyVaultControlled bool
	if buySupplyVault != nil {
		soldCollat, newDebt, isBuyVaultControlled = withdrawAssets(
			amtOut, buySupplyVault.EulerAccountAssets, buySupplyVault.IsControllerEnabled)
	} else {
		soldCollat = uint256.NewInt(0)
		newDebt = uint256.NewInt(0)
		isBuyVaultControlled = false
	}

	// Deposit: repay borrow vault debt, then deposit to supply vault
	sellDebt := uint256.NewInt(0)
	if sellBorrowVault != nil && sellBorrowVault.Debt != nil {
		sellDebt = sellBorrowVault.Debt
	}
	var vaultDepositAmt, reserveDepositAmt, repayAmt *uint256.Int
	var isSellVaultControlled bool
	if sellSupplyVault != nil {
		vaultDepositAmt, reserveDepositAmt, repayAmt, _, isSellVaultControlled = p.depositAssets(
			amtIn, p.getFee(zeroForOne), sellDebt,
			sellSupplyVault.IsControllerEnabled)
	} else {
		vaultDepositAmt = uint256.NewInt(0)
		reserveDepositAmt = uint256.NewInt(0)
		repayAmt = uint256.NewInt(0)
		isSellVaultControlled = false
	}

	newCollat := uint256.NewInt(0)
	if strings.EqualFold(debtVaultAddr, sellBorrowVaultAddr) {
		if vaultDepositAmt.Lt(debt) {
			if newDebt.Sign() > 0 {
				return nil, shared.ErrMultiDebts
			}
			debt.Sub(debt, vaultDepositAmt)
		} else {
			newCollat.Sub(vaultDepositAmt, debt)
			debt.Clear()
		}
	} else {
		newCollat = vaultDepositAmt
	}

	if newDebt.Sign() > 0 {
		if !strings.EqualFold(debtVaultAddr, buyBorrowVaultAddr) {
			if debt.Sign() > 0 {
				return nil, shared.ErrMultiDebts
			}
			debtVaultIdx = buyVaultIdx
		}
		debt.Add(debt, newDebt)
	}

	// Solvency check
	collatVal := p.collateralValue.Clone()
	if debt.Sign() > 0 {
		debtVault := p.BorrowVault[debtVaultIdx]
		if debtVault == nil {
			return nil, shared.ErrInsolvency
		}

		valuePrices, ltvs := debtVault.ValuePrices, debtVault.LTVs

		var liabilityVal, tmp uint256.Int
		liabilityVal.Mul(debt, debtVault.DebtPrice).Div(&liabilityVal, big256.UBasisPoint)

		if collatVal.IsZero() && p.Collaterals != nil {
			for i, collateral := range p.Collaterals {
				if i < len(ltvs) && i < len(valuePrices) && valuePrices[i] != nil {
					collatVal.Add(collatVal, tmp.Mul(tmp.Mul(collateral, tmp.SetUint64(ltvs[i])), valuePrices[i]))
				}
			}
		}

		vaultValuePrices, vaultLtvs := debtVault.VaultValuePrices, debtVault.VaultLTVs
		if vaultValuePrices[sellVaultIdx] != nil {
			// In solvency check, newCollat is what was actually deposited (vaultDepositAmt)
			collatVal.Add(collatVal,
				tmp.Mul(tmp.Mul(newCollat, tmp.SetUint64(vaultLtvs[sellVaultIdx])), vaultValuePrices[sellVaultIdx]))
		}

		if vaultValuePrices[buyVaultIdx] != nil {
			collatVal.Sub(collatVal,
				tmp.Mul(tmp.Mul(soldCollat, tmp.SetUint64(vaultLtvs[buyVaultIdx])), vaultValuePrices[buyVaultIdx]))
		}

		// Apply a safety buffer (99.9%) to the collateral value for swap limit checks
		collatValWithBuffer, _ := tmp.MulDivOverflow(collatVal, shared.BufferSwapLimit, big256.UBasisPoint)
		if liabilityVal.Gt(collatValWithBuffer) {
			return nil, shared.ErrInsolvency
		}
	}

	var newReserve0, newReserve1 uint256.Int
	if zeroForOne {
		newReserve0.Add(p.reserves[0], reserveDepositAmt)
		newReserve1.Sub(p.reserves[1], amtOut)
	} else {
		newReserve1.Add(p.reserves[1], reserveDepositAmt)
		newReserve0.Sub(p.reserves[0], amtOut)
	}

	if !p.verify(&newReserve0, &newReserve1) {
		return nil, shared.ErrCurveViolation
	}

	return &shared.SwapInfo{
		Reserves:              [2]*uint256.Int{&newReserve0, &newReserve1},
		WithdrawAmount:        soldCollat,
		BorrowAmount:          newDebt,
		ReserveDepositAmount:  reserveDepositAmt,
		VaultDepositAmount:    vaultDepositAmt,
		RepayAmount:           repayAmt,
		Debt:                  debt,
		DebtVaultIdx:          debtVaultIdx,
		CollateralValue:       collatVal,
		IsSellVaultControlled: isSellVaultControlled,
		IsBuyVaultControlled:  isBuyVaultControlled,
		ZeroForOne:            zeroForOne,
	}, nil
}

func (p *PoolSimulator) getFee(zeroForOne bool) *uint256.Int {
	if p.SwapHookedOps&hooks.HookGetFee != 0 && p.hook != nil {
		fee, err := p.hook.GetFee(&hooks.GetFeeParams{
			Asset0IsInput: zeroForOne,
			Reserve0:      p.reserves[0],
			Reserve1:      p.reserves[1],
		})
		if err == nil {
			return uint256.NewInt(fee)
		}
	}

	if zeroForOne {
		return p.Fee0
	}
	return p.Fee1
}

func (p *PoolSimulator) computeQuote(amount *uint256.Int, isExactIn, isZeroForOne bool) (*uint256.Int, error) {
	if amount.IsZero() {
		return uint256.NewInt(0), nil
	}

	fee := p.getFee(isZeroForOne)
	if fee.Cmp(big256.BONE) >= 0 {
		return nil, shared.ErrSwapRejected
	}

	inLimit, outLimit, err := p.calcLimits(isZeroForOne, fee)
	if err != nil {
		return nil, err
	}

	var effAmount = new(uint256.Int).Set(amount)
	if isExactIn {
		feeAmt, _ := new(uint256.Int).MulDivOverflow(amount, fee, big256.BONE)
		effAmount.Sub(amount, feeAmt)
	}

	quote, err := p.findCurvePoint(effAmount, isExactIn, isZeroForOne)
	if err != nil {
		return nil, err
	}

	if isExactIn {
		if effAmount.Gt(inLimit) || quote.Gt(outLimit) {
			return nil, shared.ErrSwapLimitExceeded
		}
	} else {
		if effAmount.Gt(outLimit) || quote.Gt(inLimit) {
			return nil, shared.ErrSwapLimitExceeded
		}
		var denominator uint256.Int
		denominator.Sub(big256.BONE, fee)
		quote.Mul(quote, big256.BONE)
		quote.Div(quote, &denominator)
	}

	return quote, nil
}

func (p *PoolSimulator) calcLimits(isZeroForOne bool, fee *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	var inLimit, outLimit uint256.Int

	inLimit.Set(shared.MaxUint112)
	outLimit.Set(shared.MaxUint112)

	supplyIn := lo.Ternary(isZeroForOne, p.SupplyVault[0], p.SupplyVault[1])
	borrowIn := lo.Ternary(isZeroForOne, p.BorrowVault[0], p.BorrowVault[1])

	// Supply caps on input: maxDeposit + debt(if borrow vault exists)
	maxDeposit := new(uint256.Int).Set(supplyIn.MaxDeposit)
	if borrowIn != nil && borrowIn.Debt != nil {
		maxDeposit.Add(maxDeposit, borrowIn.Debt)
	}
	if maxDeposit.Lt(&inLimit) {
		inLimit.Set(maxDeposit)
	}

	var reserveLimit uint256.Int
	if isZeroForOne {
		if p.reserves[1].Gt(p.MinReserve1) {
			reserveLimit.Sub(p.reserves[1], p.MinReserve1)
		}
	} else {
		if p.reserves[0].Gt(p.MinReserve0) {
			reserveLimit.Sub(p.reserves[0], p.MinReserve0)
		}
	}
	if reserveLimit.Lt(&outLimit) {
		outLimit.Set(&reserveLimit)
	}

	supplyOut := lo.Ternary(isZeroForOne, p.SupplyVault[1], p.SupplyVault[0])
	borrowOut := lo.Ternary(isZeroForOne, p.BorrowVault[1], p.BorrowVault[0])

	supplyBalance := uint256.NewInt(0)
	if supplyOut != nil && supplyOut.EulerAccountAssets != nil {
		supplyBalance = supplyOut.EulerAccountAssets
	}

	if borrowOut != nil {
		supplyCash := supplyOut.Cash
		if supplyCash == nil {
			supplyCash = uint256.NewInt(0)
		}

		isSameVault := (isZeroForOne && p.SupplyVault1 == p.BorrowVault1) ||
			(!isZeroForOne && p.SupplyVault0 == p.BorrowVault0)

		if supplyBalance.Gt(supplyCash) || isSameVault {
			if supplyCash.Lt(&outLimit) {
				outLimit.Set(supplyCash)
			}
		} else {
			cashLimit := new(uint256.Int).Set(supplyBalance)
			if borrowOut.Cash != nil {
				cashLimit.Add(cashLimit, borrowOut.Cash)
			}
			if cashLimit.Lt(&outLimit) {
				outLimit.Set(cashLimit)
			}
		}

		if borrowOut.BorrowCap != nil && !borrowOut.BorrowCap.Eq(big256.UMax) {
			totalBorrows := borrowOut.TotalBorrows
			if totalBorrows == nil {
				totalBorrows = uint256.NewInt(0)
			}

			var maxWithdraw uint256.Int
			maxWithdraw.Set(supplyBalance)
			if totalBorrows.Lt(borrowOut.BorrowCap) {
				var remaining uint256.Int
				remaining.Sub(borrowOut.BorrowCap, totalBorrows)
				maxWithdraw.Add(&maxWithdraw, &remaining)
			}
			if maxWithdraw.Lt(&outLimit) {
				outLimit.Set(&maxWithdraw)
			}
		}
	} else {
		if supplyOut.Cash != nil && supplyOut.Cash.Lt(&outLimit) {
			outLimit.Set(supplyOut.Cash)
		}
	}

	{
		inLimit2, err := p.findCurvePoint(&outLimit, false, isZeroForOne)
		if err == nil && inLimit2.Cmp(shared.MaxUint112) <= 0 {
			var denominator uint256.Int
			denominator.Sub(big256.BONE, fee)
			inLimit2.Mul(inLimit2, big256.BONE)
			inLimit2.Div(inLimit2, &denominator)

			if inLimit2.Lt(&inLimit) {
				inLimit.Set(inLimit2)
			}
		} else {
			amountWithFee := new(uint256.Int)
			amountWithFee.Mul(&inLimit, new(uint256.Int).Sub(big256.BONE, fee))
			amountWithFee.Div(amountWithFee, big256.BONE)

			outLimit2, err := p.findCurvePoint(amountWithFee, true, isZeroForOne)
			if err == nil && outLimit2.Lt(&outLimit) {
				outLimit.Set(outLimit2)

				inLimit2, err = p.findCurvePoint(&outLimit, false, isZeroForOne)
				if err == nil {
					var denominator uint256.Int
					denominator.Sub(big256.BONE, fee)
					inLimit2.Mul(inLimit2, big256.BONE)
					inLimit2.Div(inLimit2, &denominator)
					if inLimit2.Lt(&inLimit) {
						inLimit.Set(inLimit2)
					}
				}
			}
		}
	}

	if outLimit.Sign() > 0 {
		outLimit.SubUint64(&outLimit, 1)
	}

	return &inLimit, &outLimit, nil
}

func (p *PoolSimulator) verify(newReserve0, newReserve1 *uint256.Int) bool {
	if newReserve0.Gt(shared.MaxUint112) || newReserve1.Gt(shared.MaxUint112) {
		return false
	}

	if newReserve0.Lt(p.MinReserve0) || newReserve1.Lt(p.MinReserve1) {
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
		yNew, err = _F(newReserve1, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0, p.ConcentrationY)
		if err != nil {
			return false
		}
		return newReserve0.Cmp(yNew) >= 0
	}

	if newReserve1.Lt(p.EquilibriumReserve1) {
		return false
	}
	yNew, err = _F(newReserve0, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1, p.ConcentrationX)
	if err != nil {
		return false
	}
	return newReserve1.Cmp(yNew) >= 0
}

func (p *PoolSimulator) findCurvePoint(amount *uint256.Int, isExactIn bool, isZeroForOne bool) (*uint256.Int, error) {
	output := new(uint256.Int)

	if isExactIn {
		if isZeroForOne {
			xNew := new(uint256.Int).Add(p.reserves[0], amount)
			var yNew *uint256.Int
			var err error

			if xNew.Cmp(p.EquilibriumReserve0) <= 0 {
				yNew, err = _F(xNew, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1, p.ConcentrationX)
			} else {
				yNew, err = _FInverse(xNew, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0, p.ConcentrationY)
			}
			if err != nil {
				return nil, err
			}
			if p.reserves[1].Gt(yNew) {
				output.Sub(p.reserves[1], yNew)
				return output, nil
			}
			return uint256.NewInt(0), nil
		}

		yNew := new(uint256.Int).Add(p.reserves[1], amount)
		var xNew *uint256.Int
		var err error

		if yNew.Cmp(p.EquilibriumReserve1) <= 0 {
			xNew, err = _F(yNew, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0, p.ConcentrationY)
		} else {
			xNew, err = _FInverse(yNew, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1, p.ConcentrationX)
		}
		if err != nil {
			return nil, err
		}
		if p.reserves[0].Gt(xNew) {
			output.Sub(p.reserves[0], xNew)
			return output, nil
		}
		return uint256.NewInt(0), nil
	}

	if isZeroForOne {
		if !p.reserves[1].Gt(amount) {
			return nil, shared.ErrInsufficientReserve
		}
		yNew := new(uint256.Int).Sub(p.reserves[1], amount)
		var xNew *uint256.Int
		var err error

		if yNew.Cmp(p.EquilibriumReserve1) <= 0 {
			xNew, err = _F(yNew, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0, p.ConcentrationY)
		} else {
			xNew, err = _FInverse(yNew, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1, p.ConcentrationX)
		}
		if err != nil {
			return nil, err
		}
		if xNew.Gt(p.reserves[0]) {
			output.Sub(xNew, p.reserves[0])
			return output, nil
		}
		return uint256.NewInt(0), nil
	}

	if !p.reserves[0].Gt(amount) {
		return nil, shared.ErrInsufficientReserve
	}
	xNew := new(uint256.Int).Sub(p.reserves[0], amount)
	var yNew *uint256.Int
	var err error

	if xNew.Cmp(p.EquilibriumReserve0) <= 0 {
		yNew, err = _F(xNew, p.PriceX, p.PriceY, p.EquilibriumReserve0, p.EquilibriumReserve1, p.ConcentrationX)
	} else {
		yNew, err = _FInverse(xNew, p.PriceY, p.PriceX, p.EquilibriumReserve1, p.EquilibriumReserve0, p.ConcentrationY)
	}
	if err != nil {
		return nil, err
	}
	if yNew.Gt(p.reserves[1]) {
		output.Sub(yNew, p.reserves[1])
		return output, nil
	}

	return uint256.NewInt(0), nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.reserves = [2]*uint256.Int{p.reserves[0].Clone(), p.reserves[1].Clone()}
	cloned.collateralValue = p.collateralValue.Clone()
	cloned.SupplyVault = [2]*shared.VaultState(lo.Map(p.SupplyVault[:], func(v *shared.VaultState, _ int) *shared.VaultState {
		if v == nil {
			return nil
		}
		clonedVault := *v
		return &clonedVault
	}))
	cloned.BorrowVault = [3]*shared.VaultState(lo.Map(p.BorrowVault[:], func(v *shared.VaultState, _ int) *shared.VaultState {
		if v == nil {
			return nil
		}
		clonedVault := *v
		return &clonedVault
	}))
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(*shared.SwapInfo)
	if !ok {
		return
	}

	p.reserves[0].Set(swapInfo.Reserves[0])
	p.reserves[1].Set(swapInfo.Reserves[1])

	from, to := 0, 1
	if !swapInfo.ZeroForOne {
		from, to = to, from
	}

	vaultDepositAmt, repayAmt := swapInfo.VaultDepositAmount, swapInfo.RepayAmount
	sellVault := p.SupplyVault[from]
	sellBorrowVault := p.BorrowVault[from]

	sellVault.IsControllerEnabled = swapInfo.IsSellVaultControlled
	supplyDepositAmt := shared.SubTill0(vaultDepositAmt, repayAmt)
	sellVault.Cash = new(uint256.Int).Add(sellVault.Cash, supplyDepositAmt)
	sellVault.MaxDeposit = shared.SubTill0(sellVault.MaxDeposit, supplyDepositAmt)
	sellVault.EulerAccountAssets = new(uint256.Int).Add(sellVault.EulerAccountAssets, supplyDepositAmt)

	if sellBorrowVault != nil {
		sellBorrowVault.Cash = new(uint256.Int).Add(sellBorrowVault.Cash, repayAmt)
		sellBorrowVault.Debt = shared.SubTill0(sellBorrowVault.Debt, repayAmt)
		sellBorrowVault.TotalBorrows = shared.SubTill0(sellBorrowVault.TotalBorrows, repayAmt)
	}

	withdrawAmt, borrowAmt := swapInfo.WithdrawAmount, swapInfo.BorrowAmount
	buyVault := p.SupplyVault[to]
	buyBorrowVault := p.BorrowVault[to]

	buyVault.IsControllerEnabled = swapInfo.IsBuyVaultControlled
	buyVault.Cash = shared.SubTill0(buyVault.Cash, withdrawAmt)
	if buyVault.MaxDeposit != nil && !buyVault.MaxDeposit.Eq(big256.UMax) {
		buyVault.MaxDeposit = new(uint256.Int).Add(buyVault.MaxDeposit, withdrawAmt)
	}
	buyVault.EulerAccountAssets = shared.SubTill0(buyVault.EulerAccountAssets, withdrawAmt)

	if buyBorrowVault != nil {
		buyBorrowVault.Cash = shared.SubTill0(buyBorrowVault.Cash, borrowAmt)
		buyBorrowVault.Debt = new(uint256.Int).Add(buyBorrowVault.Debt, borrowAmt)
		buyBorrowVault.TotalBorrows = new(uint256.Int).Add(buyBorrowVault.TotalBorrows, borrowAmt)
	}

	if swapInfo.DebtVaultIdx < 2 {
		p.ControllerVault = lo.Ternary(swapInfo.DebtVaultIdx == 0, p.BorrowVault0, p.BorrowVault1)
	}

	for i := 0; i < 3; i++ {
		if p.BorrowVault[i] != nil {
			if i == swapInfo.DebtVaultIdx {
				p.BorrowVault[i].Debt.Set(swapInfo.Debt)
				p.BorrowVault[i].IsControllerEnabled = !swapInfo.Debt.IsZero()
			} else {
				p.BorrowVault[i].Debt.Clear()
				p.BorrowVault[i].IsControllerEnabled = false
			}
		}
	}

	p.collateralValue = swapInfo.CollateralValue
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolExtra{
		BlockNumber: p.Info.BlockNumber,
	}
}

func withdrawAssets(amount, balance *uint256.Int, isControllerEnabled bool) (soldCollat, newDebt *uint256.Int, _ bool) {
	if amount.Cmp(balance) <= 0 {
		return amount, big256.U0, isControllerEnabled
	}
	return balance, new(uint256.Int).Sub(amount, balance), true
}

func (p *PoolSimulator) depositAssets(
	amount,
	fee,
	debt *uint256.Int,
	isControllerEnabled bool,
) (vaultDeposit, reserveDeposit, repaid, feeAmount *uint256.Int, _ bool) {
	if amount.IsZero() {
		return big256.U0, big256.U0, big256.U0, big256.U0, isControllerEnabled
	}

	feeAmount, _ = new(uint256.Int).MulDivOverflow(amount, fee, big256.BONE)
	reserveDeposit = new(uint256.Int).Sub(amount, feeAmount)

	vaultDeposit = amount.Clone()
	if p.FeeRecipient != "" {
		vaultDeposit.Sub(vaultDeposit, feeAmount)
	}

	remainingAmount := reserveDeposit.Clone()
	repaid = uint256.NewInt(0)

	if isControllerEnabled && debt != nil {
		if remainingAmount.Gt(debt) {
			repaid.Set(debt)
		} else {
			repaid.Set(remainingAmount)
		}

		remainingAmount.Sub(remainingAmount, repaid)

		if debt.Eq(repaid) {
			isControllerEnabled = false
		}
	}

	return vaultDeposit, reserveDeposit, repaid, feeAmount, isControllerEnabled
}
