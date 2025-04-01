package eulerswap

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type Gas struct {
	Swap int64
}

type PoolSimulator struct {
	pool.Pool

	gas Gas

	feeMultiplier *uint256.Int

	equilibriumReserve0 *uint256.Int
	equilibriumReserve1 *uint256.Int

	priceX *uint256.Int
	priceY *uint256.Int

	concentrationX *uint256.Int
	concentrationY *uint256.Int

	vault0 Vault
	vault1 Vault
}

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrInvalidReserve = errors.New("invalid reserve")
	ErrInvalidAmount  = errors.New("invalid amount")
	ErrOverflow       = errors.New("math overflow")
	ErrCurveViolation = errors.New("curve violation")
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var originalReserves uniswapv2.ReserveData
	if err := json.Unmarshal([]byte(entityPool.Extra), &originalReserves); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		gas: defaultGas,
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
		return nil, ErrInvalidAmount
	}

	amountOut, swapInfo, err := s.swap(true, indexIn == 0, amountIn)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:            s.gas.Swap,
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
		return nil, ErrInvalidAmount
	}

	amountOut, swapInfo, err := s.swap(false, indexIn == 0, amountOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountOut.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:           s.gas.Swap,
		SwapInfo:      swapInfo,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		if swapInfo.NewReserve0 != nil {
			s.Info.Reserves[0] = swapInfo.NewReserve0.ToBig()
		}
		if swapInfo.NewReserve0 != nil {
			s.Info.Reserves[1] = swapInfo.NewReserve1.ToBig()
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return uniswapv2.PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) swap(exactIn, asset0IsInput bool, amountIn *uint256.Int) (*uint256.Int, SwapInfo, error) {
	reserve0, overflow := uint256.FromBig(s.Pool.Info.Reserves[0])
	if overflow {
		return nil, SwapInfo{}, ErrInvalidReserve
	}

	reserve1, overflow := uint256.FromBig(s.Pool.Info.Reserves[1])
	if overflow {
		return nil, SwapInfo{}, ErrInvalidReserve
	}

	amountOut, err := s.computeQuote(exactIn, asset0IsInput, reserve0, reserve1, amountIn)
	if err != nil {
		return nil, SwapInfo{}, err
	}

	if asset0IsInput {
		reserve0.Add(reserve0, amountIn)
		reserve1.Sub(reserve1, amountOut)
	} else {
		reserve0.Sub(reserve0, amountOut)
		reserve1.Add(reserve1, amountIn)
	}

	if !s.verify(reserve0, reserve1) {
		return nil, SwapInfo{}, ErrCurveViolation
	}

	return amountOut, SwapInfo{
		NewReserve0: reserve0,
		NewReserve1: reserve1,
	}, nil
}

func (s *PoolSimulator) computeQuote(exactIn, asset0IsInput bool, reserve0, reserve1, amountIn *uint256.Int) (*uint256.Int, error) {
	inLimit, outLimit := s.calcLimits(asset0IsInput, reserve0, reserve1)

	quote, err := BinarySearch(reserve0, reserve1, amountIn, exactIn, asset0IsInput, s.verify)
	if err != nil {
		return nil, err
	}

	if exactIn {
		if amountIn.Gt(inLimit) || quote.Gt(outLimit) {
			return nil, ErrSwapLimitExceeded
		}

		return quote, nil
	}

	if amountIn.Gt(outLimit) || quote.Gt(inLimit) {
		return nil, ErrSwapLimitExceeded
	}

	quote.Mul(quote, oneE18)
	quote.Add(quote, new(uint256.Int).Sub(s.feeMultiplier, big256.One))
	return quote.Div(quote, s.feeMultiplier), nil
}

func (s *PoolSimulator) verify(newReserve0, newReserve1 *uint256.Int) bool {
	if newReserve0.Cmp(maxUint112) > 0 || newReserve1.Cmp(maxUint112) > 0 {
		return false
	}

	if newReserve0.Cmp(s.equilibriumReserve0) >= 0 {
		if newReserve1.Cmp(s.equilibriumReserve1) >= 0 {
			return true
		}

		result, err := f(
			newReserve1,
			s.priceY,
			s.priceX,
			s.equilibriumReserve1,
			s.equilibriumReserve0,
			s.concentrationY,
		)
		if err != nil {
			return false
		}

		return newReserve0.Cmp(result) >= 0
	} else {
		if newReserve1.Cmp(s.equilibriumReserve1) < 0 {
			return false
		}

		result, err := f(
			newReserve0,
			s.priceX,
			s.priceY,
			s.equilibriumReserve0,
			s.equilibriumReserve1,
			s.concentrationX,
		)
		if err != nil {
			return false
		}

		return newReserve1.Cmp(result) >= 0
	}
}

// f implements the EulerSwap curve definition
// Pre-conditions: x <= x0, 1 <= {px,py} <= 1e36, {x0,y0} <= type(uint112).max, c <= 1e18
func f(
	x, px, py, x0, y0, c *uint256.Int,
) (*uint256.Int, error) {
	t1 := new(uint256.Int)
	t2 := new(uint256.Int)

	t1.Sub(x0, x)

	t1.Mul(px, t1)

	t2.Mul(c, x)

	t2.Add(t2, new(uint256.Int).Mul(new(uint256.Int).Sub(oneE18, c), x0))

	t1.Mul(t1, t2)

	t2.Mul(x, oneE18)

	t1.Add(t1, new(uint256.Int).Sub(t2, big256.One))

	t1.Div(t1, t2)

	if t1.Cmp(MaxUint248) > 0 {
		return nil, ErrOverflow
	}

	t1.Add(t1, new(uint256.Int).Sub(py, big256.One))
	t1.Div(t1, py)

	return new(uint256.Int).Add(y0, t1), nil
}

func (s *PoolSimulator) calcLimits(asset0IsInput bool, reserve0, reserve1 *uint256.Int) (*uint256.Int, *uint256.Int) {
	var (
		outLimit     = new(uint256.Int)
		inLimit      = new(uint256.Int)
		cash         = new(uint256.Int)
		maxWithdraw  = new(uint256.Int)
		totalBorrows = new(uint256.Int)
		vaultBalance = new(uint256.Int)
	)

	if asset0IsInput {
		inLimit.Add(s.vault0.Debt, s.vault0.MaxDeposit)
		outLimit.Set(reserve1)

		cash = s.vault1.Cash

		if s.vault1.Cash.Lt(outLimit) {
			outLimit.Set(cash)
		}

		maxWithdraw.Set(s.vault1.MaxWithdraw)

		totalBorrows = s.vault1.TotalBorrows

		vaultBalance = s.vault1.Balance
	} else {
		inLimit.Add(s.vault1.Debt, s.vault1.MaxDeposit)
		outLimit.Set(reserve0)

		cash = s.vault0.Cash

		maxWithdraw.Set(s.vault0.MaxWithdraw)

		totalBorrows = s.vault0.TotalBorrows

		vaultBalance = s.vault0.Balance
	}

	if totalBorrows.Gt(maxWithdraw) {
		maxWithdraw.SetUint64(0)
	} else {
		maxWithdraw.Sub(maxWithdraw, totalBorrows)
	}

	if maxWithdraw.Gt(cash) {
		maxWithdraw.Set(cash)
	}

	maxWithdraw.Add(maxWithdraw, vaultBalance)

	if maxWithdraw.Lt(outLimit) {
		outLimit.Set(maxWithdraw)
	}

	return inLimit, outLimit
}
