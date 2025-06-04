package uniswapv2

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	fee          *uint256.Int
	feePrecision *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
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
		fee:          uint256.NewInt(extra.Fee),
		feePrecision: uint256.NewInt(extra.FeePrecision),
	}, nil
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

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	if reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)
	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	// NOTE: Intentionally comment out, since kAfter should always smaller than kBefore.
	// balanceIn := new(uint256.Int).Add(reserveIn, amountIn)
	// balanceOut := new(uint256.Int).Sub(reserveOut, amountOut)

	// balanceInAdjusted := new(uint256.Int).Sub(
	// 	new(uint256.Int).Mul(balanceIn, s.feePrecision),
	// 	new(uint256.Int).Mul(amountIn, s.fee),
	// )
	// balanceOutAdjusted := new(uint256.Int).Mul(balanceOut, s.feePrecision)

	// kBefore := new(uint256.Int).Mul(new(uint256.Int).Mul(reserveIn, reserveOut), new(uint256.Int).Mul(s.feePrecision, s.feePrecision))
	// kAfter := new(uint256.Int).Mul(balanceInAdjusted, balanceOutAdjusted)

	// if kAfter.Cmp(kBefore) < 0 {
	// 	return nil, ErrInvalidK
	// }

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas: defaultGas + extraGasByExchange[s.GetExchange()],
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
		return nil, ErrInsufficientOutputAmount
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	if reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountIn, err := s.getAmountIn(amountOut, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	}

	if amountIn.Cmp(reserveIn) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	balanceIn := new(uint256.Int).Add(reserveIn, amountIn)
	balanceOut := new(uint256.Int).Sub(reserveOut, amountOut)

	balanceInAdjusted := balanceIn.Sub(
		balanceIn.Mul(balanceIn, s.feePrecision),
		amountOut.Mul(amountIn, s.fee),
	)
	balanceOutAdjusted := balanceOut.Mul(balanceOut, s.feePrecision)

	kBefore := reserveIn.Mul(reserveIn.Mul(reserveIn, reserveOut),
		reserveOut.Mul(s.feePrecision, s.feePrecision))
	kAfter := balanceInAdjusted.Mul(balanceInAdjusted, balanceOutAdjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return nil, ErrInvalidK
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas: defaultGas + extraGasByExchange[s.GetExchange()],
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Pool.Info.Reserves[indexIn] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:             s.fee.Uint64(),
		FeePrecision:    s.feePrecision.Uint64(),
		BlockNumber:     s.Pool.Info.BlockNumber,
		ApprovalAddress: approvalAddressByExchange[s.GetExchange()],
	}
}

func (s *PoolSimulator) getAmountOut(amountIn, reserveIn, reserveOut *uint256.Int) *uint256.Int {
	var numerator, denominator uint256.Int
	amountInWithFee := numerator.Mul(amountIn, numerator.Sub(s.feePrecision, s.fee))
	denominator.Add(denominator.Mul(reserveIn, s.feePrecision), amountInWithFee)
	numerator.Mul(amountInWithFee, reserveOut)
	return numerator.Div(&numerator, &denominator)
}

func (s *PoolSimulator) getAmountIn(amountOut, reserveIn, reserveOut *uint256.Int) (amountIn *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}
	}()

	numerator := SafeMul(
		SafeMul(reserveIn, amountOut),
		s.feePrecision,
	)
	denominator := SafeMul(
		SafeSub(reserveOut, amountOut),
		SafeSub(s.feePrecision, s.fee),
	)

	return SafeAdd(numerator.Div(numerator, denominator), number.Number_1), nil
}
