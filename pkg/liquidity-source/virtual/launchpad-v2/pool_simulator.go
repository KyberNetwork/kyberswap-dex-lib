package launchpadv2

import (
	"fmt"
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

var u99 = uint256.NewInt(99)

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

	antiSniperBuyTaxStartValue *uint256.Int

	taxStartTime int64
	startTime    int64
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
		Pool: pool.Pool{pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
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
	}

	return p, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
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
		isBuy                    bool
		amountOut                *uint256.Int
		newBalanceA, newBalanceB *uint256.Int
		newReserveA, newReserveB *uint256.Int
		gas                      int64
	)

	if indexIn == 0 {
		var amountOutBeforeFee *uint256.Int

		amountOutBeforeFee, amountOut = s.sell(amountIn)
		if amountOut.Cmp(balanceB) >= 0 {
			return nil, ErrInsufficientOutputAmount
		}

		newBalanceA = new(uint256.Int).Add(balanceA, amountIn)
		newBalanceB = new(uint256.Int).Sub(balanceB, amountOut)

		newReserveA = new(uint256.Int).Add(s.reserveA, amountIn)
		newReserveB = new(uint256.Int).Sub(s.reserveB, amountOutBeforeFee)

		gas = defaultSellGas

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

		isBuy = true
		gas = defaultBuyGas
	}

	if newReserveA.Cmp(s.gradThreshold) <= 0 &&
		s.calculateAntiSniperTax(time.Now().Unix()).IsZero() {
		gas += defaultOpenTradingOnUniswapGas
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:            gas,
		SwapInfo: SwapInfo{
			isBuy:       isBuy,
			Bonding:     s.bonding,
			newReserveA: newReserveA,
			newReserveB: newReserveB,
			newBalanceA: newBalanceA,
			newBalanceB: newBalanceB,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
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
		isBuy                    bool
		amountInNeeded           *uint256.Int
		newBalanceA, newBalanceB *uint256.Int
		newReserveA, newReserveB *uint256.Int
		err                      error
		gas                      int64
	)

	if indexOut == 1 {
		if amountOut.Cmp(balanceB) >= 0 {
			return nil, ErrInsufficientOutputAmount
		}

		var amountOutBeforeFee *uint256.Int

		amountInNeeded, amountOutBeforeFee, err = s.sellExactOut(amountOut)
		if err != nil {
			return nil, err
		}

		newBalanceA = new(uint256.Int).Add(balanceA, amountInNeeded)
		newBalanceB = new(uint256.Int).Sub(balanceB, amountOut)

		newReserveA = new(uint256.Int).Add(s.reserveA, amountInNeeded)
		newReserveB = new(uint256.Int).Sub(s.reserveB, amountOutBeforeFee)

		gas = defaultSellGas
	} else {
		if amountOut.Cmp(balanceA) >= 0 {
			return nil, ErrInsufficientOutputAmount
		}

		amountInNeeded, err = s.buyExactOut(amountOut)
		if err != nil {
			return nil, err
		}

		newBalanceA = new(uint256.Int).Sub(balanceA, amountOut)
		newBalanceB = new(uint256.Int).Add(balanceB, amountInNeeded)

		newReserveA = new(uint256.Int).Sub(s.reserveA, amountOut)
		newReserveB = new(uint256.Int).Add(s.reserveB, amountInNeeded)

		isBuy = true
		gas = defaultBuyGas
	}

	if newReserveA.Cmp(s.gradThreshold) <= 0 && s.calculateAntiSniperTax(time.Now().Unix()).IsZero() {
		gas += defaultOpenTradingOnUniswapGas
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: amountInNeeded.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: bignumber.ZeroBI},
		Gas:           gas,
		SwapInfo: SwapInfo{
			isBuy:       isBuy,
			Bonding:     s.bonding,
			newReserveA: newReserveA,
			newReserveB: newReserveB,
			newBalanceA: newBalanceA,
			newBalanceB: newBalanceB,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok && len(s.GetReserves()) == 2 {
		s.reserveA = swapInfo.newReserveA
		s.reserveB = swapInfo.newReserveB
		s.Info.Reserves[0] = swapInfo.newBalanceA.ToBig()
		s.Info.Reserves[1] = swapInfo.newBalanceB.ToBig()
	}
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

	elapsed := now - finalStart
	taxReduction := uint256.NewInt(uint64(elapsed / 60)) // 1% per minute

	if s.antiSniperBuyTaxStartValue.Cmp(taxReduction) <= 0 {
		return uint256.NewInt(0)
	}

	return new(uint256.Int).Sub(s.antiSniperBuyTaxStartValue, taxReduction)
}

func (s *PoolSimulator) effectiveBuyTax(now int64) *uint256.Int {
	antiSniperTax := s.calculateAntiSniperTax(now)
	totalTax := new(uint256.Int).Add(s.buyTax, antiSniperTax)

	// Cap at 99% so user gets at least 1%
	if totalTax.Cmp(u99) > 0 {
		return new(uint256.Int).Set(u99)
	}

	return totalTax
}

func (s *PoolSimulator) sell(amountIn *uint256.Int) (*uint256.Int, *uint256.Int) {
	amountOut := s.getAmountsOut(amountIn, false)

	fee := s.sellTax

	txFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(fee, amountOut),
		u256.U100,
	)

	amountOutAfterFee := new(uint256.Int).Sub(amountOut, txFee)

	return amountOut, amountOutAfterFee
}

func (s *PoolSimulator) buy(amountIn *uint256.Int) (*uint256.Int, *uint256.Int) {
	fee := s.effectiveBuyTax(time.Now().Unix())

	txFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(fee, amountIn),
		u256.U100,
	)

	amountInAfterFee := new(uint256.Int).Sub(amountIn, txFee)

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
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}
	}()

	amountOutBeforeFee = new(uint256.Int).Div(
		new(uint256.Int).Mul(amountOut, u256.U100),
		new(uint256.Int).Sub(u256.U100, s.sellTax),
	)

	newReserveB := new(uint256.Int).Sub(s.reserveB, amountOutBeforeFee)
	newReserveA := new(uint256.Int).Div(s.kLast, newReserveB)
	amountIn = new(uint256.Int).Sub(newReserveA, s.reserveA)

	return
}

func (s *PoolSimulator) buyExactOut(amountOut *uint256.Int) (amountInBeforeFee *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}
	}()

	newReserveA := new(uint256.Int).Sub(s.reserveA, amountOut)
	newReserveB := new(uint256.Int).Div(s.kLast, newReserveA)
	amountIn := new(uint256.Int).Sub(newReserveB, s.reserveB)

	fee := s.effectiveBuyTax(time.Now().Unix())

	amountInBeforeFee = new(uint256.Int).Div(
		new(uint256.Int).Mul(amountIn, u256.U100),
		new(uint256.Int).Sub(u256.U100, fee),
	)

	return
}
