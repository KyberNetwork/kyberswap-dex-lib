package fourmeme

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidReserve          = errors.New("invalid reserve")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInvalidAmountOut        = errors.New("invalid amount out")
	ErrTokenTxFee              = errors.New("tokenTxFee must be empty to swap")
	ErrTradingHalted           = errors.New("trading is halted")
	ErrTradingDisabled         = errors.New("trading is disabled")
	ErrTokenNotLaunched        = errors.New("token has not been launched yet")
	ErrFundsTooLow             = errors.New("funds too low")
	ErrInvalidTradeAmount      = errors.New("error: Amount mod gwei != 0")
	ErrSpendingFundsTooMuch    = errors.New("spending too much funds")
	ErrSmallOrderSize          = errors.New("order size is too small")
	ErrPriceTooLow             = errors.New("price is too low")
)

type PoolSimulator struct {
	pool.Pool
	gas Gas

	tradingHalted   bool
	tradingDisabled bool
	launchTime      int64
	minTradeFee     *uint256.Int
	tradingFeeRate  *uint256.Int
	tokenTxFee      *uint256.Int

	f3  *uint256.Int
	f10 *uint256.Int
	f11 *uint256.Int
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

	if amountIn.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	var (
		amountOut *uint256.Int
		err       error
	)

	// is Buy
	if indexIn == 0 {
		offers, overflow := uint256.FromBig(s.Pool.Info.Reserves[1])
		if overflow {
			return nil, ErrInvalidReserve
		}

		amountOut, err = s.buyTokenExactIn(amountIn, offers)
	} else {
		amountOut, err = s.sellToken(amountIn)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: ZERO.ToBig()},
		Gas:            s.gas.Swap,
		SwapInfo: SwapInfo{
			TradedAmount: amountIn,
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

	if indexOut == 0 {
		return nil, errors.New("this DEX doesn't support sell exact out")
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	if amountOut.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	offers, overflow := uint256.FromBig(s.Pool.Info.Reserves[1])
	if overflow {
		return nil, ErrInvalidReserve
	}

	amountIn, err := s.buyTokenExactOut(offers, amountOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: ZERO.ToBig()},
		Gas:           s.gas.Swap,
		SwapInfo: SwapInfo{
			TradedAmount: amountIn,
		},
	}, nil

}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		if s.GetTokenIndex(params.TokenAmountIn.Token) == 0 {
			s.f11 = new(uint256.Int).Sub(s.f11, swapInfo.TradedAmount)
		} else {
			s.f11 = new(uint256.Int).Add(s.f11, swapInfo.TradedAmount)
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) buyTokenExactOut(offers, buyAmount *uint256.Int) (*uint256.Int, error) {
	return s.buyToken(ZERO, buyAmount, offers)
}

func (s *PoolSimulator) buyTokenExactIn(funds, offers *uint256.Int) (*uint256.Int, error) {
	return s.buyToken(funds, ZERO, offers)
}

func (s *PoolSimulator) sellToken(sellAmount *uint256.Int) (amountOut *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}
	}()

	if s.tradingHalted {
		return nil, ErrTradingHalted
	}

	if time.Now().Unix() < s.launchTime {
		return nil, ErrTokenNotLaunched
	}

	if s.tradingDisabled {
		return nil, ErrTradingDisabled
	}

	if new(uint256.Int).Mod(sellAmount, EXP9).Sign() != 0 {
		return nil, ErrInvalidTradeAmount
	}

	if _, overflow := new(uint256.Int).AddOverflow(s.f11, sellAmount); overflow {
		return nil, number.ErrOverflow
	}

	fund := calcSellCost(s.f10, s.f11, sellAmount)

	tradingFee := calcTradingFee(fund, s.tradingFeeRate)
	if fund.Cmp(tradingFee) <= 0 {
		return nil, ErrSmallOrderSize
	}

	if s.f3.Sign() != 0 && fund.Lt(s.f3) {
		return nil, ErrPriceTooLow
	}

	return new(uint256.Int).Sub(fund, tradingFee), nil
}

func calcSellCost(varg0, varg1, varg2 *uint256.Int) *uint256.Int {
	v0 := number.SafeMul(varg0, EXP18)
	v1 := number.SafeDiv(v0, varg1)
	v2 := number.SafeAdd(varg1, varg2)
	v2 = number.SafeDiv(v0, v2)

	return number.SafeSub(v1, v2)
}

func (s *PoolSimulator) buyToken(funds, offers, buyAmount *uint256.Int) (amountOut *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}
	}()

	if s.tradingHalted {
		return nil, ErrTradingHalted
	}

	if s.tokenTxFee.Sign() > 0 {
		return nil, ErrTokenTxFee
	}

	if time.Now().Unix() < s.launchTime {
		return nil, ErrTokenNotLaunched
	}

	if s.tradingDisabled {
		return nil, ErrTradingDisabled
	}

	var tradingFee *uint256.Int
	if buyAmount.Sign() != 0 {
		if funds.Sign() > 0 {
			if funds.Cmp(s.minTradeFee) <= 0 {
				return nil, ErrFundsTooLow
			}

			fundsAfterFee := number.SafeAdd(s.tradingFeeRate, PRECISION)
			fundsAfterFee = number.SafeDiv(new(uint256.Int).Mul(funds, PRECISION), fundsAfterFee)

			tradingFee = number.SafeSub(funds, fundsAfterFee)
			if tradingFee.Lt(s.minTradeFee) {
				fundsAfterFee = number.SafeSub(funds, s.minTradeFee)
			}

			buyAmount = calcBuyAmount(s.f10, s.f11, fundsAfterFee)
		}
	}

	if new(uint256.Int).Mod(buyAmount, EXP9).Sign() != 0 {
		return nil, ErrInvalidTradeAmount
	}

	if buyAmount.Gt(offers) {
		buyAmount = new(uint256.Int).Set(offers)
	}

	cost := calcBuyCost(s.f10, s.f11, buyAmount)

	tradingFee = calcTradingFee(cost, s.tradingFeeRate)

	return amountOut.Add(tradingFee, cost), nil
}

func calcBuyCost(varg0, varg1, varg2 *uint256.Int) *uint256.Int {
	v0 := number.SafeMul(varg0, EXP18)
	v1 := number.SafeDiv(v0, varg1)
	v2 := number.SafeSub(varg1, varg2)
	v2 = number.SafeDiv(v0, v2)

	return number.SafeSub(v2, v1)
}

func calcTradingFee(tradeLiquidity, lastLiquidityTradedEMA *uint256.Int) *uint256.Int {
	return number.SafeDiv(number.SafeMul(tradeLiquidity, lastLiquidityTradedEMA), PRECISION)
}

func calcBuyAmount(amount, maxFunds, fundsAfterFee *uint256.Int) *uint256.Int {
	v0 := number.SafeMul(amount, EXP18)
	v1 := number.SafeDiv(v0, maxFunds)
	v1 = number.SafeAdd(v1, fundsAfterFee)
	v0 = number.SafeDiv(v0, v1)

	buyAmount := number.SafeSub(maxFunds, v0)

	return number.SafeSub(buyAmount, new(uint256.Int).Mod(buyAmount, EXP9))
}
