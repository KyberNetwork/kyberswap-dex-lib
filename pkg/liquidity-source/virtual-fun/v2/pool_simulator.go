package v2

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	trading, launchExecuted bool

	buyTax  *uint256.Int
	sellTax *uint256.Int

	reserveA *uint256.Int
	reserveB *uint256.Int

	kLast           *uint256.Int
	bonding, router string

	gradThreshold *uint256.Int
	graduated     bool

	antiSniperBuyTaxStartValue *uint256.Int

	taxStartTime int64
	startTime    int64
	isXLaunch    bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	err := json.Unmarshal([]byte(ep.Extra), &extra)
	if err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	err = json.Unmarshal([]byte(ep.StaticExtra), &staticExtra)
	if err != nil {
		return nil, err
	}

	p := &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     ep.Address,
				Exchange:    ep.Exchange,
				Type:        ep.Type,
				Tokens:      lo.Map(ep.Tokens, func(a *entity.PoolToken, _ int) string { return a.Address }),
				Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
				BlockNumber: ep.BlockNumber,
			},
		},
		trading:                    extra.Trading,
		launchExecuted:             extra.LaunchExecuted,
		buyTax:                     uint256.MustFromBig(extra.BuyTax),
		sellTax:                    uint256.MustFromBig(extra.SellTax),
		kLast:                      uint256.MustFromBig(extra.KLast),
		reserveA:                   uint256.MustFromBig(extra.ReserveA),
		reserveB:                   uint256.MustFromBig(extra.ReserveB),
		gradThreshold:              uint256.MustFromBig(extra.GradThreshold),
		bonding:                    staticExtra.Bonding,
		router:                     staticExtra.Router,
		antiSniperBuyTaxStartValue: uint256.MustFromBig(extra.AntiSniperBuyTaxStartValue),
		taxStartTime:               extra.TaxStartTime.Int64(),
		startTime:                  extra.StartTime.Int64(),
		graduated:                  extra.Graduated,
		isXLaunch:                  extra.IsXLaunch,
	}

	return p, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.graduated {
		return nil, ErrTokenGraduated
	}

	if !s.trading || !s.launchExecuted {
		return nil, ErrInvalidTokenStatus
	}

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

	if amountIn.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	balanceA, overflow := uint256.FromBig(s.Info.Reserves[0])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceB, overflow := uint256.FromBig(s.Info.Reserves[1])
	if overflow {
		return nil, ErrInvalidReserve
	}

	var (
		isBuy                    = indexIn != 0
		amountOut                *uint256.Int
		newBalanceA, newBalanceB *uint256.Int
		newReserveA, newReserveB *uint256.Int
		fee                      *uint256.Int
		graduated                bool
	)

	if !isBuy {
		var amountOutBeforeFee *uint256.Int

		amountOutBeforeFee, amountOut = s.sell(amountIn)
		if amountOutBeforeFee.Cmp(balanceB) >= 0 {
			return nil, ErrInsufficientOutputAmount
		}

		newBalanceA = new(uint256.Int).Add(balanceA, amountIn)
		newBalanceB = new(uint256.Int).Sub(balanceB, amountOutBeforeFee)

		newReserveA = new(uint256.Int).Add(s.reserveA, amountIn)
		newReserveB = new(uint256.Int).Sub(s.reserveB, amountOutBeforeFee)

		fee = new(uint256.Int).Sub(amountOutBeforeFee, amountOut)
	} else {
		var amountInAfterFee *uint256.Int

		amountInAfterFee, amountOut = s.buy(amountIn)
		if amountOut.Cmp(balanceA) >= 0 {
			return nil, ErrInsufficientOutputAmount
		}

		newBalanceA = new(uint256.Int).Sub(balanceA, amountOut)
		newBalanceB = new(uint256.Int).Add(balanceB, amountInAfterFee)

		newReserveA = new(uint256.Int).Sub(s.reserveA, amountOut)
		newReserveB = new(uint256.Int).Add(s.reserveB, amountInAfterFee)

		fee = new(uint256.Int).Sub(amountIn, amountInAfterFee)
	}

	gas := lo.Ternary(isBuy, defaultBuyGas, defaultSellGas)
	if newReserveA.Cmp(s.gradThreshold) <= 0 &&
		s.calculateAntiSniperTax(time.Now().Unix()).IsZero() {
		gas += defaultOpenTradingOnUniswapGas
		graduated = true
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: lo.Ternary(isBuy, tokenAmountIn.Token, tokenOut), Amount: fee.ToBig()},
		Gas:            gas,
		SwapInfo: SwapInfo{
			Bonding:     s.bonding,
			newReserveA: newReserveA,
			newReserveB: newReserveB,
			newBalanceA: newBalanceA,
			newBalanceB: newBalanceB,
			graduated:   graduated,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if s.graduated {
		return nil, ErrTokenGraduated
	}

	if !s.trading || !s.launchExecuted {
		return nil, ErrInvalidTokenStatus
	}

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

	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	balanceA, overflow := uint256.FromBig(s.Info.Reserves[0])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceB, overflow := uint256.FromBig(s.Info.Reserves[1])
	if overflow {
		return nil, ErrInvalidReserve
	}

	var (
		isBuy                    = indexIn != 0
		amountInNeeded           *uint256.Int
		newBalanceA, newBalanceB *uint256.Int
		newReserveA, newReserveB *uint256.Int
		fee                      *uint256.Int
		graduated                bool
		err                      error
	)

	if !isBuy {
		var amountOutBeforeFee *uint256.Int

		amountInNeeded, amountOutBeforeFee, err = s.sellExactOut(amountOut)
		if err != nil {
			return nil, err
		}

		if amountOutBeforeFee.Cmp(balanceB) >= 0 {
			return nil, ErrInsufficientOutputAmount
		}

		newBalanceA = new(uint256.Int).Add(balanceA, amountInNeeded)
		newBalanceB = new(uint256.Int).Sub(balanceB, amountOutBeforeFee)

		newReserveA = new(uint256.Int).Add(s.reserveA, amountInNeeded)
		newReserveB = new(uint256.Int).Sub(s.reserveB, amountOutBeforeFee)

		fee = new(uint256.Int).Sub(amountOutBeforeFee, amountOut)
	} else {
		if amountOut.Cmp(balanceA) >= 0 {
			return nil, ErrInsufficientOutputAmount
		}

		var amountInAfterFee *uint256.Int
		amountInNeeded, amountInAfterFee, err = s.buyExactOut(amountOut)
		if err != nil {
			return nil, err
		}

		newBalanceA = new(uint256.Int).Sub(balanceA, amountOut)
		newBalanceB = new(uint256.Int).Add(balanceB, amountInAfterFee)

		newReserveA = new(uint256.Int).Sub(s.reserveA, amountOut)
		newReserveB = new(uint256.Int).Add(s.reserveB, amountInAfterFee)

		fee = new(uint256.Int).Sub(amountInNeeded, amountInAfterFee)
	}

	gas := lo.Ternary(isBuy, defaultBuyGas, defaultSellGas)
	if newReserveA.Cmp(s.gradThreshold) <= 0 &&
		s.calculateAntiSniperTax(time.Now().Unix()).IsZero() {
		gas += defaultOpenTradingOnUniswapGas
		graduated = true
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountInNeeded.ToBig()},
		Fee:           &pool.TokenAmount{Token: lo.Ternary(isBuy, tokenIn, tokenAmountOut.Token), Amount: fee.ToBig()},
		Gas:           gas,
		SwapInfo: SwapInfo{
			Bonding:     s.bonding,
			newReserveA: newReserveA,
			newReserveB: newReserveB,
			newBalanceA: newBalanceA,
			newBalanceB: newBalanceB,
			graduated:   graduated,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		s.reserveA = swapInfo.newReserveA
		s.reserveB = swapInfo.newReserveB
		s.Info.Reserves = []*big.Int{swapInfo.newBalanceA.ToBig(), swapInfo.newBalanceB.ToBig()}
		s.graduated = swapInfo.graduated
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		ApprovalAddress: s.router,
	}
}

func (s *PoolSimulator) calculateAntiSniperTax(now int64) *uint256.Int {
	finalStart := s.startTime
	if s.taxStartTime > 0 {
		finalStart = s.taxStartTime
	}

	if now <= finalStart {
		return new(uint256.Int).Set(s.antiSniperBuyTaxStartValue)
	}

	elapsed := now + 12 - finalStart
	taxReduction := uint256.NewInt(uint64(lo.Ternary(s.isXLaunch, elapsed, elapsed/60)))

	if s.antiSniperBuyTaxStartValue.Cmp(taxReduction) <= 0 {
		return u256.New0()
	}

	return new(uint256.Int).Sub(s.antiSniperBuyTaxStartValue, taxReduction)
}

func (s *PoolSimulator) cappedAntiSniperTax(now int64) *uint256.Int {
	antiSniperTax := s.calculateAntiSniperTax(now)

	sum := new(uint256.Int).Add(s.buyTax, antiSniperTax)
	if sum.Cmp(u256.U99) > 0 {
		return new(uint256.Int).Sub(u256.U99, s.buyTax)
	}

	return antiSniperTax
}

func (s *PoolSimulator) sell(amountIn *uint256.Int) (*uint256.Int, *uint256.Int) {
	amountOut := s.getAmountsOut(amountIn, false)

	txFee, _ := new(uint256.Int).MulDivOverflow(s.sellTax, amountOut, u256.U100)

	amountOutAfterFee := new(uint256.Int).Sub(amountOut, txFee)

	return amountOut, amountOutAfterFee
}

func (s *PoolSimulator) buy(amountIn *uint256.Int) (*uint256.Int, *uint256.Int) {
	now := time.Now().Unix()
	antiSniperTax := s.cappedAntiSniperTax(now)

	normalTxFee, _ := new(uint256.Int).MulDivOverflow(s.buyTax, amountIn, u256.U100)
	antiSniperTxFee, _ := new(uint256.Int).MulDivOverflow(antiSniperTax, amountIn, u256.U100)

	amountInAfterFee := new(uint256.Int).Sub(amountIn, normalTxFee)
	amountInAfterFee.Sub(amountInAfterFee, antiSniperTxFee)

	amountOut := s.getAmountsOut(amountInAfterFee, true)

	return amountInAfterFee, amountOut
}

func (s *PoolSimulator) getAmountsOut(amountIn *uint256.Int, isBuy bool) *uint256.Int {
	var amountOut = new(uint256.Int)

	if isBuy {
		newReserveB := new(uint256.Int).Add(s.reserveB, amountIn)
		newReserveA := new(uint256.Int).Div(s.kLast, newReserveB)
		amountOut = amountOut.Sub(s.reserveA, newReserveA)
	} else {
		newReserveA := new(uint256.Int).Add(s.reserveA, amountIn)
		newReserveB := new(uint256.Int).Div(s.kLast, newReserveA)
		amountOut = amountOut.Sub(s.reserveB, newReserveB)
	}

	return amountOut
}

func (s *PoolSimulator) sellExactOut(amountOut *uint256.Int) (amountIn, amountOutBeforeFee *uint256.Int, err error) {
	amountOutBeforeFee = new(uint256.Int).Div(
		new(uint256.Int).Mul(amountOut, u256.U100),
		new(uint256.Int).Sub(u256.U100, s.sellTax),
	)

	newReserveB := new(uint256.Int).Sub(s.reserveB, amountOutBeforeFee)
	newReserveA := new(uint256.Int).Div(s.kLast, newReserveB)
	amountIn = new(uint256.Int).Sub(newReserveA, s.reserveA)

	return
}

func (s *PoolSimulator) buyExactOut(amountOut *uint256.Int) (amountInBeforeFee, amountInAfterFee *uint256.Int, err error) {
	newReserveA := new(uint256.Int).Sub(s.reserveA, amountOut)
	newReserveB := new(uint256.Int).Div(s.kLast, newReserveA)
	amountInAfterFee = new(uint256.Int).Sub(newReserveB, s.reserveB)

	now := time.Now().Unix()
	antiSniperTax := s.cappedAntiSniperTax(now)
	totalTax := new(uint256.Int).Add(s.buyTax, antiSniperTax)

	amountInBeforeFee = new(uint256.Int).Div(
		new(uint256.Int).Mul(amountInAfterFee, u256.U100),
		new(uint256.Int).Sub(u256.U100, totalTax),
	)

	return
}
