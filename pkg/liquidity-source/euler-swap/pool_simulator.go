package eulerswap

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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

	pause         uint32 // 0 = unactivated, 1 = unlocked, 2 = locked
	feeMultiplier *uint256.Int

	equilibriumReserve0, equilibriumReserve1 *uint256.Int

	priceX, priceY *uint256.Int

	concentrationX, concentrationY *uint256.Int

	vault0, vault1 string
	eulerAccount   string

	vaults []Vault
}

var (
	ErrInvalidVaults  = errors.New("invalid vaults")
	ErrInvalidToken   = errors.New("invalid token")
	ErrInvalidReserve = errors.New("invalid reserve")
	ErrInvalidAmount  = errors.New("invalid amount")
	ErrSwapIsPaused   = errors.New("swap is paused")
	ErrOverflow       = errors.New("math overflow")
	ErrCurveViolation = errors.New("curve violation")
)

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

	if extra.Vaults == nil {
		extra.Vaults = make([]Vault, 2)
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
		vaults:              extra.Vaults,
		pause:               extra.Pause,
		vault0:              staticExtra.Vault0,
		vault1:              staticExtra.Vault1,
		eulerAccount:        staticExtra.EulerAccount,
		feeMultiplier:       staticExtra.FeeMultiplier,
		equilibriumReserve0: staticExtra.EquilibriumReserve0,
		equilibriumReserve1: staticExtra.EquilibriumReserve1,
		priceX:              staticExtra.PriceX,
		priceY:              staticExtra.PriceY,
		concentrationX:      staticExtra.ConcentrationX,
		concentrationY:      staticExtra.ConcentrationY,
		gas:                 defaultGas,
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

	if s.pause != 1 {
		return nil, ErrSwapIsPaused
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

	if s.pause != 1 {
		return nil, ErrSwapIsPaused
	}

	amountIn, swapInfo, err := s.swap(false, indexIn == 0, amountOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
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

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolExtra{
		Vault0:              s.vault0,
		Vault1:              s.vault1,
		EulerAccount:        s.eulerAccount,
		EquilibriumReserve0: s.equilibriumReserve0.ToBig(),
		EquilibriumReserve1: s.equilibriumReserve1.ToBig(),
		FeeMultiplier:       s.feeMultiplier.ToBig(),
		PriceY:              s.priceY.ToBig(),
		PriceX:              s.priceX.ToBig(),
		ConcentrationY:      s.concentrationY.ToBig(),
		ConcentrationX:      s.concentrationX.ToBig(),
		BlockNumber:         s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) swap(exactIn, asset0IsInput bool, amount *uint256.Int) (*uint256.Int, SwapInfo, error) {
	reserve0, overflow := uint256.FromBig(s.Pool.Info.Reserves[0])
	if overflow {
		return nil, SwapInfo{}, ErrInvalidReserve
	}

	reserve1, overflow := uint256.FromBig(s.Pool.Info.Reserves[1])
	if overflow {
		return nil, SwapInfo{}, ErrInvalidReserve
	}

	quote, err := s.computeQuote(exactIn, asset0IsInput, reserve0, reserve1, amount)
	if err != nil {
		return nil, SwapInfo{}, err
	}

	amountIn, amountOut := amount, quote
	if !exactIn {
		amountIn, amountOut = quote, amount
	}

	if exactIn {
		if asset0IsInput {
			reserve0.Add(reserve0, amountIn)
			reserve1.Sub(reserve1, amountOut)
		} else {
			reserve0.Sub(reserve0, amountOut)
			reserve1.Add(reserve1, amountIn)
		}
	}

	if !s.verify(reserve0, reserve1) {
		return nil, SwapInfo{}, ErrCurveViolation
	}

	return quote, SwapInfo{
		NewReserve0: reserve0,
		NewReserve1: reserve1,
	}, nil
}

func (s *PoolSimulator) computeQuote(exactIn, asset0IsInput bool, reserve0, reserve1, amount *uint256.Int) (*uint256.Int, error) {
	inLimit, outLimit := s.calcLimits(asset0IsInput, reserve0, reserve1)

	quote, err := BinarySearch(reserve0, reserve1, amount, exactIn, asset0IsInput, s.verify)
	if err != nil {
		return nil, err
	}

	if exactIn {
		if amount.Gt(inLimit) || quote.Gt(outLimit) {
			return nil, ErrSwapLimitExceeded
		}

		return quote, nil
	}

	if amount.Gt(outLimit) || quote.Gt(inLimit) {
		return nil, ErrSwapLimitExceeded
	}

	quote.Mul(quote, oneE18)
	quote.Add(quote, new(uint256.Int).Sub(s.feeMultiplier, big256.One))
	return quote.Div(quote, s.feeMultiplier), nil
}

func (s *PoolSimulator) calcLimits(asset0IsInput bool, reserve0, reserve1 *uint256.Int) (*uint256.Int, *uint256.Int) {
	inLimit := new(uint256.Int)
	outLimit := new(uint256.Int)

	var vaultInIndex, vaultOutIndex int
	var reserveOut *uint256.Int

	if asset0IsInput {
		vaultInIndex = 0
		vaultOutIndex = 1
		reserveOut = reserve1
	} else {
		vaultInIndex = 1
		vaultOutIndex = 0
		reserveOut = reserve0
	}

	inLimit.Add(s.vaults[vaultInIndex].Debt, s.vaults[vaultInIndex].MaxDeposit)

	outLimit.Set(reserveOut)
	if s.vaults[vaultOutIndex].Cash.Lt(outLimit) {
		outLimit.Set(s.vaults[vaultOutIndex].Cash)
	}

	maxWithdraw := new(uint256.Int).Set(s.vaults[vaultOutIndex].MaxWithdraw)
	if s.vaults[vaultOutIndex].TotalBorrows.Gt(maxWithdraw) {
		maxWithdraw.SetUint64(0)
	} else {
		maxWithdraw.Sub(maxWithdraw, s.vaults[vaultOutIndex].TotalBorrows)
	}

	if maxWithdraw.Gt(s.vaults[vaultOutIndex].Cash) {
		maxWithdraw.Set(s.vaults[vaultOutIndex].Cash)
	}

	maxWithdraw.Add(maxWithdraw, s.vaults[vaultOutIndex].EulerAccountAssets)

	if maxWithdraw.Lt(outLimit) {
		outLimit.Set(maxWithdraw)
	}

	return inLimit, outLimit
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
