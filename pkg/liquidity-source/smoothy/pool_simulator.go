package smoothy

import (
	"math/big"
	"slices"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool
	Extra
}

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(pt *entity.PoolToken, _ int) string { return pt.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		Extra: extra,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenOut

	inIdx, outIdx := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if inIdx < 0 || outIdx < 0 {
		return nil, ErrInvalidToken
	}

	amountIn := params.TokenAmountIn.Amount
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrZeroSwap
	}

	swapInfo, err := p.getSwapAmount(inIdx, outIdx, number.SetFromBig(amountIn))
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: swapInfo.amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: swapInfo.fee.ToBig(),
		},
		SwapInfo: swapInfo,
		Gas:      defaultGas,
	}, nil
}

func (p *PoolSimulator) getSwapAmount(idxIn, idxOut int, amountIn *uint256.Int) (*SwapInfo, error) {
	infoIn := p.TokenInfos[idxIn]
	infoOut := p.TokenInfos[idxOut]

	totalBalance := p.TotalBalance.Clone()
	tidInBalance := new(uint256.Int).Mul(infoIn.Balance, u256.TenPow(infoIn.DecimalMulitiplier))

	bTokenInAmountNormalized := new(uint256.Int).Mul(amountIn, u256.TenPow(infoIn.DecimalMulitiplier))
	sMinted, err := p.getMintAmount(
		bTokenInAmountNormalized,
		totalBalance,
		tidInBalance,
		infoIn.SoftWeight,
		infoIn.HardWeight,
	)
	if err != nil {
		return nil, err
	}

	totalBalance.Add(totalBalance, bTokenInAmountNormalized)
	tidInBalance.Add(tidInBalance, bTokenInAmountNormalized)

	tidOutBalance := new(uint256.Int).Mul(infoOut.Balance, u256.TenPow(infoOut.DecimalMulitiplier))

	bTokenOutAmountNormalized, err := p.redeemFindOne(
		tidOutBalance,
		totalBalance,
		tidInBalance,
		sMinted,
		infoIn.SoftWeight,
		infoIn.HardWeight,
	)
	if err != nil {
		return nil, err
	}

	bTokenOutAmount := new(uint256.Int).Div(bTokenOutAmountNormalized, u256.TenPow(infoOut.DecimalMulitiplier))

	fee, _ := new(uint256.Int).MulDivOverflow(bTokenOutAmount, p.SwapFee, u256.BONE)
	adminFee, _ := new(uint256.Int).MulDivOverflow(fee, p.AdminFeePct, u256.BONE)
	bTokenOutAmount.Sub(bTokenOutAmount, fee)

	if bTokenOutAmount.Gt(infoOut.Balance) {
		return nil, ErrInsufficientBalance
	}

	return &SwapInfo{
		IdxIn:     idxIn,
		IdxOut:    idxOut,
		amountIn:  amountIn,
		amountOut: bTokenOutAmount,
		adminFee:  adminFee,
		fee:       fee,
	}, nil
}

func (p *PoolSimulator) getMintAmount(bTokenAmountNormalized, oldBalance, oldTokenBalance, softWeight,
	hardWeight *uint256.Int) (*uint256.Int, error) {

	newBalance := new(uint256.Int).Add(oldBalance, bTokenAmountNormalized)
	newTokenBalance := new(uint256.Int).Add(oldTokenBalance, bTokenAmountNormalized)

	temp0 := new(uint256.Int).Mul(newTokenBalance, u256.BONE)
	temp1 := new(uint256.Int).Mul(softWeight, newBalance)
	if temp0.Cmp(temp1) <= 0 {
		return bTokenAmountNormalized.Clone(), nil
	}

	temp1.Mul(hardWeight, newBalance)
	if temp0.Gt(temp1) {
		return nil, ErrMintNewPercentageExceedsHardWeight
	}

	s := uint256.NewInt(0)
	if !temp0.Mul(oldTokenBalance, u256.BONE).Gt(temp1.Mul(softWeight, oldBalance)) {
		temp0.Sub(temp0.Mul(oldBalance, softWeight), temp1.Mul(oldTokenBalance, u256.BONE))
		temp1.Sub(u256.BONE, softWeight)
		s.Div(temp0, temp1)
	}

	temp0.MulDivOverflow(newBalance, u256.BONE, temp1.Add(oldBalance, s))
	ldelta, err := p.log(temp0)
	if err != nil {
		return nil, err
	}

	t := new(uint256.Int).Sub(oldBalance, oldTokenBalance)
	t.Mul(t, ldelta)
	t.Sub(t, temp0.Mul(temp0.Sub(bTokenAmountNormalized, s), temp1.Sub(u256.BONE, hardWeight)))
	t.Div(t, temp0.Sub(hardWeight, softWeight))
	s.Add(s, t)

	if s.Gt(bTokenAmountNormalized) {
		return nil, ErrNegativePenalty
	}

	return s, nil
}

func (p *PoolSimulator) redeemFindOne(tidOutBalance, totalBalance, tidInBalance, sTokenAmount, softWeight,
	hardWeight *uint256.Int) (*uint256.Int, error) {
	temp := u256.MulDiv(tidOutBalance, u256.U999, u256.U1000)
	redeemAmountNormalized := u256.Min(sTokenAmount, temp).Clone()

	for i := 0; i < 256; i++ {
		penalty, err := p.redeemPenaltyFor(
			totalBalance,
			tidInBalance,
			redeemAmountNormalized,
			softWeight,
			hardWeight,
		)
		if err != nil {
			return nil, err
		}

		sNeeded := new(uint256.Int).Add(redeemAmountNormalized, penalty)

		var fx uint256.Int
		if sNeeded.Gt(sTokenAmount) {
			fx.Sub(sNeeded, sTokenAmount)
		} else {
			fx.Sub(sTokenAmount, sNeeded)
		}

		if fx.Lt(temp.Div(redeemAmountNormalized, u256.U100000)) {
			if redeemAmountNormalized.Gt(sTokenAmount) {
				return nil, ErrOutAmountGreaterThanLPAmount
			}

			if redeemAmountNormalized.Gt(tidOutBalance) {
				return nil, ErrInsufficientBalance
			}

			return redeemAmountNormalized, nil
		}

		dfx, err := p.redeemPenaltyDerivativeForOne(
			totalBalance,
			tidInBalance,
			redeemAmountNormalized,
			softWeight,
			hardWeight,
		)
		if err != nil {
			return nil, err
		}

		temp.MulDivOverflow(&fx, u256.BONE, dfx)
		if sNeeded.Gt(sTokenAmount) {
			redeemAmountNormalized.Sub(redeemAmountNormalized, temp)
		} else {
			redeemAmountNormalized.Add(redeemAmountNormalized, temp)
		}
	}

	return nil, ErrCannotFindProperResolutionOfFX
}

func (p *PoolSimulator) redeemPenaltyFor(totalBalance, tokenBalance, redeemAmount, softWeight,
	hardWeight *uint256.Int) (*uint256.Int, error) {
	newTotalBalance := new(uint256.Int).Sub(totalBalance, redeemAmount)

	temp0 := new(uint256.Int).Mul(tokenBalance, u256.BONE)
	temp1 := new(uint256.Int).Mul(newTotalBalance, softWeight)

	if !temp0.Gt(temp1) {
		return u256.New0(), nil
	}

	temp1.Mul(newTotalBalance, hardWeight)
	if temp0.Gt(temp1) {
		return nil, ErrRedeemHardLimitWeightBroken
	}

	bx := u256.New0()
	totalSoft := new(uint256.Int).Mul(totalBalance, softWeight)
	tokenScaled := new(uint256.Int).Mul(tokenBalance, u256.BONE)

	if totalSoft.Gt(tokenScaled) {
		temp := new(uint256.Int).Div(tokenScaled, softWeight)
		bx = new(uint256.Int).Sub(totalBalance, temp)
	}

	temp0.Sub(redeemAmount, bx)
	temp0.Mul(temp0, hardWeight)

	temp1.Mul(hardWeight, newTotalBalance)
	temp1.Div(temp1, u256.BONE)
	temp1.Sub(temp1, tokenBalance)

	logArg := new(uint256.Int).Div(temp0, temp1)
	logArg.Add(logArg, u256.BONE)

	logVal, err := p.log(logArg)
	if err != nil {
		return nil, err
	}

	temp1.Mul(tokenBalance, logVal)

	temp0.Sub(hardWeight, softWeight)
	temp0.Mul(temp0, temp1)
	temp0.Div(temp0, hardWeight)
	temp0.Div(temp0, hardWeight)

	temp1.Sub(redeemAmount, bx)
	temp1.Mul(temp1, softWeight)
	temp1.Div(temp1, hardWeight)

	return temp0.Sub(temp0, temp1), nil
}

func (p *PoolSimulator) redeemPenaltyDerivativeForOne(totalBalance, tokenBalance, redeemAmount, softWeight,
	hardWeight *uint256.Int) (*uint256.Int, error) {
	newTotalBalance := new(uint256.Int).Sub(totalBalance, redeemAmount)

	temp0 := new(uint256.Int).Mul(tokenBalance, u256.BONE)
	temp1 := new(uint256.Int).Mul(newTotalBalance, softWeight)
	if !temp0.Gt(temp1) {
		return u256.BONE.Clone(), nil
	}

	temp0.Sub(temp0, temp1)

	temp1.Mul(hardWeight, newTotalBalance)
	temp1.Div(temp1, u256.BONE)
	temp1.Sub(temp1, tokenBalance)

	temp1.Div(temp0, temp1)
	temp1.Add(temp1, u256.BONE)

	return temp1, nil
}

func (p *PoolSimulator) log(x *uint256.Int) (*uint256.Int, error) {
	if x.Lt(u256.BONE) {
		return nil, ErrLogXInvalidInput
	}

	maxVal := new(uint256.Int).Lsh(u256.BONE, 63)
	if x.Cmp(maxVal) >= 0 {
		return nil, ErrLogXTooLarge
	}

	if x.Cmp(logUpperBound) <= 0 {
		return p.logApprox(x)
	}

	xx := new(big.Int).Lsh(x.ToBig(), 64)
	xx.Div(xx, u256.BONE.ToBig())

	yy, err := p.lg2(xx)
	if err != nil {
		return nil, err
	}

	result := new(big.Int).Mul(yy, ln2Multiplier.ToBig())
	result.Rsh(result, 64)

	y, overflow := uint256.FromBig(result)
	if overflow {
		return nil, number.ErrOverflow
	}

	return y, nil
}

func (p *PoolSimulator) logApprox(x *uint256.Int) (*uint256.Int, error) {
	if x.Cmp(u256.BONE) < 0 {
		return nil, ErrLogApproxXMustGteOne
	}

	z := new(uint256.Int).Sub(x, u256.BONE)
	temp := new(uint256.Int)

	result := z.Clone()

	zPow := new(uint256.Int).Mul(z, z)
	zPow.Div(zPow, u256.BONE)

	temp.Div(zPow, u256.U2)
	result.Sub(result, temp)

	zPow.Mul(zPow, z)
	zPow.Div(zPow, u256.BONE)

	temp.Div(zPow, u256.U3)
	result.Add(result, temp)

	zPow.Mul(zPow, z)
	zPow.Div(zPow, u256.BONE)

	temp.Div(zPow, u256.U4)
	result.Sub(result, temp)

	zPow.Mul(zPow, z)
	zPow.Div(zPow, u256.BONE)

	temp.Div(zPow, u256.U5)
	result.Add(result, temp)

	return result, nil
}

func (p *PoolSimulator) lg2(x *big.Int) (*big.Int, error) {
	if x.Sign() <= 0 {
		return nil, ErrLg2XMustBePositive
	}

	msb := int64(0)
	xc := new(big.Int).Set(x)
	temp := new(big.Int)

	temp.SetString("10000000000000000", 16)
	if xc.Cmp(temp) >= 0 {
		xc.Rsh(xc, 64)
		msb += 64
	}

	temp.SetUint64(0x100000000)
	if xc.Cmp(temp) >= 0 {
		xc.Rsh(xc, 32)
		msb += 32
	}

	temp.SetUint64(0x10000)
	if xc.Cmp(temp) >= 0 {
		xc.Rsh(xc, 16)
		msb += 16
	}

	temp.SetUint64(0x100)
	if xc.Cmp(temp) >= 0 {
		xc.Rsh(xc, 8)
		msb += 8
	}

	temp.SetUint64(0x10)
	if xc.Cmp(temp) >= 0 {
		xc.Rsh(xc, 4)
		msb += 4
	}

	temp.SetUint64(0x4)
	if xc.Cmp(temp) >= 0 {
		xc.Rsh(xc, 2)
		msb += 2
	}

	temp.SetUint64(0x2)
	if xc.Cmp(temp) >= 0 {
		msb += 1
	}

	res := big.NewInt(msb - 64)
	res.Lsh(res, 64)

	ux := new(big.Int).Set(x)
	shiftAmount := 127 - msb
	if shiftAmount > 0 {
		ux.Lsh(ux, uint(shiftAmount))
	} else if shiftAmount < 0 {
		ux.Rsh(ux, uint(-shiftAmount))
	}

	bit := new(big.Int).SetUint64(0x8000000000000000)
	stopBit := new(big.Int).SetUint64(0x80000000000)
	val127 := big.NewInt(127)

	for bit.Cmp(stopBit) > 0 {
		ux.Mul(ux, ux)

		b := new(big.Int).Rsh(ux, 255)

		shift := new(big.Int).Add(val127, b)
		ux.Rsh(ux, uint(shift.Uint64()))

		if b.Sign() > 0 {
			temp.Mul(bit, b)
			res.Add(res, temp)
		}

		bit.Rsh(bit, 1)
	}

	return res, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(*SwapInfo)
	if !ok {
		return
	}

	infoIn := &p.TokenInfos[swapInfo.IdxIn]
	infoOut := &p.TokenInfos[swapInfo.IdxOut]

	infoIn.Balance.Add(infoIn.Balance, swapInfo.amountIn)

	totalOut := new(uint256.Int).Add(swapInfo.amountOut, swapInfo.adminFee)
	infoOut.Balance.Sub(infoOut.Balance, new(uint256.Int).Add(swapInfo.amountOut, swapInfo.adminFee))

	inNormalized := new(uint256.Int).Mul(swapInfo.amountIn, u256.TenPow(infoIn.DecimalMulitiplier))
	outNormalized := new(uint256.Int).Mul(totalOut, u256.TenPow(infoOut.DecimalMulitiplier))

	p.TotalBalance.Add(p.TotalBalance, inNormalized)
	p.TotalBalance.Sub(p.TotalBalance, outNormalized)
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{BlockNumber: p.Info.BlockNumber}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.TokenInfos = slices.Clone(p.TokenInfos)
	for i := range cloned.TokenInfos {
		cloned.TokenInfos[i].Balance = cloned.TokenInfos[i].Balance.Clone()
	}
	cloned.TotalBalance = p.TotalBalance.Clone()

	return &cloned
}
