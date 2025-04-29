package virtualfun

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	gas Gas

	buyTax  *uint256.Int
	sellTax *uint256.Int

	reserveA *uint256.Int
	reserveB *uint256.Int

	kLast          *uint256.Int
	bondingAddress string

	gradThreshold *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	err := json.Unmarshal([]byte(entityPool.Extra), &extra)
	if err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	err = json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra)
	if err != nil {
		return nil, err
	}

	p := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},

		buyTax:         uint256.MustFromBig(extra.BuyTax),
		sellTax:        uint256.MustFromBig(extra.SellTax),
		kLast:          uint256.MustFromBig(extra.KLast),
		reserveA:       uint256.MustFromBig(extra.ReserveA),
		reserveB:       uint256.MustFromBig(extra.ReserveB),
		gradThreshold:  uint256.MustFromBig(extra.GradThreshold),
		bondingAddress: staticExtra.BondingAddress,

		gas: defaultGas,
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

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	balanceA, overflow := uint256.FromBig(s.Pool.Info.Reserves[0])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceB, overflow := uint256.FromBig(s.Pool.Info.Reserves[1])
	if overflow {
		return nil, ErrInvalidReserve
	}

	var (
		isBuy                    bool
		amountOut                *uint256.Int
		newBalanceA, newBalanceB *uint256.Int
		newReserveA, newReserveB *uint256.Int
	)

	// Swap from token X to VIRTUAL
	if indexIn == 0 {
		var amountOutBeforeFee *uint256.Int

		amountOutBeforeFee, amountOut = s.sell(amountIn)
		if amountOut.Cmp(balanceB) >= 0 {
			return nil, ErrInsufficientOutputAmount
		}

		newBalanceA = new(uint256.Int).Add(balanceA, amountIn)
		newBalanceB = new(uint256.Int).Sub(balanceA, amountOut)

		newReserveA = new(uint256.Int).Add(s.reserveA, amountIn)
		newReserveB = new(uint256.Int).Sub(s.reserveB, amountOutBeforeFee)

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
	}

	gas := s.gas.Swap
	if newReserveA.Cmp(s.gradThreshold) <= 0 {
		gas += bondingCurveApplicationGas
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:            gas,
		SwapInfo: SwapInfo{
			IsBuy:          isBuy,
			BondingAddress: s.bondingAddress,
			TokenAddress:   s.Pool.Info.Tokens[0],
			NewReserveA:    newReserveA,
			NewReserveB:    newReserveB,
			NewBalanceA:    newBalanceA,
			NewBalanceB:    newBalanceB,
		},
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

	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	balanceA, overflow := uint256.FromBig(s.Pool.Info.Reserves[0])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceB, overflow := uint256.FromBig(s.Pool.Info.Reserves[1])
	if overflow {
		return nil, ErrInvalidReserve
	}

	var (
		isBuy                    bool
		amountInNeeded           *uint256.Int
		newBalanceA, newBalanceB *uint256.Int
		newReserveA, newReserveB *uint256.Int
		err                      error
	)

	// Swap from token X to VIRTUAL
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
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountInNeeded.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: bignumber.ZeroBI},
		Gas:           s.gas.Swap,
		SwapInfo: SwapInfo{
			IsBuy:          isBuy,
			BondingAddress: s.bondingAddress,
			TokenAddress:   s.Pool.Info.Tokens[0],
			NewReserveA:    newReserveA,
			NewReserveB:    newReserveB,
			NewBalanceA:    newBalanceA,
			NewBalanceB:    newBalanceB,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok && len(s.Pool.GetReserves()) == 2 {
		s.reserveA = swapInfo.NewReserveA
		s.reserveB = swapInfo.NewReserveB
		s.Pool.Info.Reserves[0] = swapInfo.NewBalanceA.ToBig()
		s.Pool.Info.Reserves[1] = swapInfo.NewBalanceB.ToBig()
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) sell(amountIn *uint256.Int) (*uint256.Int, *uint256.Int) {
	amountOut := s.getAmountsOut(amountIn, false)

	fee := s.sellTax

	txFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(fee, amountOut),
		U100,
	)

	amountOutAfterFee := new(uint256.Int).Sub(amountOut, txFee)

	return amountOut, amountOutAfterFee
}

func (s *PoolSimulator) buy(amountIn *uint256.Int) (*uint256.Int, *uint256.Int) {
	fee := s.buyTax

	txFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(fee, amountIn),
		U100,
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
		new(uint256.Int).Mul(amountOut, U100),
		new(uint256.Int).Sub(U100, s.sellTax),
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

	amountInBeforeFee = new(uint256.Int).Div(
		new(uint256.Int).Mul(amountIn, U100),
		new(uint256.Int).Sub(U100, s.buyTax),
	)

	return
}
