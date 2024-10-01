package pool

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/logger"
)

var (
	ErrCalcAmountOutPanic = errors.New("calcAmountOut was panic")
	ErrInsufficientAmount = errors.New("not enough amount")
	ErrNotConverge        = errors.New("secant loop cannot converged")
)

type Pool struct {
	Info PoolInfo
}

func (t *Pool) GetInfo() PoolInfo {
	return t.Info
}

func (t *Pool) GetTokens() []string {
	return t.Info.Tokens
}

func (t *Pool) GetReserves() []*big.Int {
	return t.Info.Reserves
}

func (t *Pool) CalculateLimit() map[string]*big.Int {
	return nil
}

// CanSwapTo is the base method to get all swappable tokens from a pool by a given token address
// Pools with custom logic should override this method
func (t *Pool) CanSwapTo(address string) []string {
	result := make([]string, 0, len(t.Info.Tokens))
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 {
		return result
	}

	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			result = append(result, t.Info.Tokens[i])
		}
	}

	return result
}

// by default pool is bi-directional so just call CanSwapTo here
// Pools with custom logic should override this method
func (t *Pool) CanSwapFrom(address string) []string {
	return t.CanSwapTo(address)
}

func (t *Pool) GetAddress() string {
	return t.Info.Address
}

func (t *Pool) GetExchange() string {
	return t.Info.Exchange
}

func (t *Pool) Equals(other IPoolSimulator) bool {
	return t.GetAddress() == other.GetAddress()
}

func (t *Pool) GetTokenIndex(address string) int {
	return t.Info.GetTokenIndex(address)
}

func (t *Pool) GetType() string {
	return t.Info.Type
}

type CalcAmountOutResult struct {
	TokenAmountOut         *TokenAmount
	Fee                    *TokenAmount
	RemainingTokenAmountIn *TokenAmount
	Gas                    int64
	SwapInfo               interface{}
}

func (r *CalcAmountOutResult) IsValid() bool {
	isRemainingValid := r.RemainingTokenAmountIn == nil || (r.RemainingTokenAmountIn != nil && r.RemainingTokenAmountIn.Amount.Sign() >= 0)
	return r.TokenAmountOut != nil && r.TokenAmountOut.Amount != nil && r.TokenAmountOut.Amount.Sign() > 0 && isRemainingValid
}

type UpdateBalanceParams struct {
	TokenAmountIn  TokenAmount
	TokenAmountOut TokenAmount
	Fee            TokenAmount
	SwapInfo       interface{}

	//Inventory is a reference to a per-request inventory balances.
	// key is tokenAddress, balance is big.Float
	// Must use reference (not copy)
	SwapLimit SwapLimit
}

type PoolToken struct {
	Token               string
	Balance             *big.Int
	Weight              uint
	PrecisionMultiplier *big.Int
	VReserve            *big.Int
}

type PoolInfo struct {
	Address     string
	ReserveUsd  float64
	SwapFee     *big.Int
	Exchange    string
	Type        string
	Tokens      []string
	Reserves    []*big.Int
	Checked     bool
	BlockNumber uint64
}

func (t *PoolInfo) GetTokenIndex(address string) int {
	for i, poolToken := range t.Tokens {
		if poolToken == address {
			return i
		}
	}
	return -1
}

type CalcAmountOutParams struct {
	TokenAmountIn TokenAmount
	TokenOut      string
	Limit         SwapLimit
}

type CalcAmountInParams struct {
	TokenAmountOut TokenAmount
	TokenIn        string
	Limit          SwapLimit
}

type CalcAmountInResult struct {
	TokenAmountIn           *TokenAmount
	RemainingTokenAmountOut *TokenAmount
	Fee                     *TokenAmount
	Gas                     int64
	SwapInfo                interface{}
}

// wrap around pool.CalcAmountOut and catch panic
func CalcAmountOut(pool IPoolSimulator, tokenAmountIn TokenAmount, tokenOut string, limit SwapLimit) (res *CalcAmountOutResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrCalcAmountOutPanic
			logger.WithFields(
				logger.Fields{
					"recover":     r,
					"poolAddress": pool.GetAddress(),
				}).Debug(err.Error())
		}
	}()

	return pool.CalcAmountOut(
		CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
			Limit:         limit,
		})
}

type ApproxAmountInParams struct {
	ExpectedTokenOut TokenAmount
	TokenIn          string
	Limit            SwapLimit
	MaxLoop          int
	Threshold        *big.Int
}

type ApproxAmountInResult struct {
	TokenAmountIn  *TokenAmount
	TokenAmountOut *TokenAmount
	Fee            *TokenAmount
	Gas            int64
	SwapInfo       interface{}
}

func ApproxAmountIn(
	pool IPoolSimulator,
	param ApproxAmountInParams,
) (*ApproxAmountInResult, error) {

	expectedTokenOut := param.ExpectedTokenOut
	expectedAmountOut := param.ExpectedTokenOut.Amount
	thresholdBI := param.Threshold

	if expectedAmountOut.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("expectedAmountOut is zero")
	}

	// if the pool support calculate directly then use it
	revSim, ok := pool.(IPoolExactOutSimulator)
	if ok {
		resIn, err := revSim.CalcAmountIn(CalcAmountInParams{
			TokenAmountOut: expectedTokenOut,
			TokenIn:        param.TokenIn,
			Limit:          param.Limit,
		})
		if err != nil {
			return nil, err
		}

		// still need to check again to see if the calculated amountIn is good enough
		resOut, err := CalcAmountOut(pool, *resIn.TokenAmountIn, expectedTokenOut.Token, nil)
		if err != nil {
			return nil, err
		}

		diff := new(big.Int).Abs(new(big.Int).Sub(resOut.TokenAmountOut.Amount, expectedAmountOut))
		if diff.Cmp(thresholdBI) < 0 {
			return &ApproxAmountInResult{
				TokenAmountIn:  resIn.TokenAmountIn,
				TokenAmountOut: resOut.TokenAmountOut,
				Fee:            resOut.Fee,
				Gas:            resOut.Gas,
				SwapInfo:       resOut.SwapInfo,
			}, nil
		}
		return nil, ErrInsufficientAmount
	}

	// otherwise try to approximate:
	// consider the pool as a function `fpool`
	// we need to find amountIn such that fpool(amountIn) = amountOut
	// in another word, we need to find root of function fpool_1 where fpool_1(x) = fpool(x) - amountOut
	// here we'll use https://en.wikipedia.org/wiki/Secant_method for that

	// get 1st initial point by converting back expectedAmountOut to tokenIn
	// this might yield error if the pool doesn't support that
	x0res, err := CalcAmountOut(pool, expectedTokenOut, param.TokenIn, param.Limit)
	if err != nil {
		logger.Debugf("error getting 1st initial point %v", err)
		return nil, err
	}
	x0 := x0res.TokenAmountOut.Amount

	// get the 2nd initial point:
	// 	- convert x0 tokenIn to tokenOut
	fx0Res, err := CalcAmountOut(pool, *x0res.TokenAmountOut, expectedTokenOut.Token, param.Limit)
	if err != nil {
		logger.Debugf("error getting 2nd initial point %v", err)
		return nil, err
	}
	if fx0Res == nil || fx0Res.TokenAmountOut.Amount.Cmp(big.NewInt(0)) == 0 {
		logger.Debugf("error getting 2nd initial point %v", fx0Res)
		return nil, ErrInsufficientAmount
	}
	// - then convert back
	x1 := new(big.Int).Div(
		new(big.Int).Mul(expectedAmountOut, x0),
		fx0Res.TokenAmountOut.Amount,
	)

	fx0 := new(big.Int).Sub(fx0Res.TokenAmountOut.Amount, expectedAmountOut)

	// Secant
	loopCount := 0
	for loopCount < param.MaxLoop {
		// fpool_1(x1) = fpool(x1) - amountOut
		fx1Res, err := CalcAmountOut(pool, TokenAmount{Token: param.TokenIn, Amount: x1}, expectedTokenOut.Token, param.Limit)
		if err != nil {
			logger.Debugf("error calculating fx1 %v", err)
			return nil, err
		}
		fx1 := new(big.Int).Sub(fx1Res.TokenAmountOut.Amount, expectedAmountOut)

		// check if we're close enough
		if new(big.Int).Abs(fx1).Cmp(thresholdBI) < 0 {
			return &ApproxAmountInResult{
				TokenAmountIn:  &TokenAmount{Token: param.TokenIn, Amount: x1},
				TokenAmountOut: fx1Res.TokenAmountOut,
				Fee:            fx1Res.Fee,
				Gas:            fx1Res.Gas,
				SwapInfo:       fx1Res.SwapInfo,
			}, nil
		}

		// if fx0 is close enough to fx1 we should stop to avoid error
		// (and because we likely won't be able to get any closer to the result)
		if fx0.Cmp(fx1) == 0 {
			logger.Debugf("breaking early, fx0=fx1=%v", fx0)
			return nil, ErrNotConverge
		}

		// get new value for x0 x1
		nom := new(big.Int).Mul(fx1, new(big.Int).Sub(x1, x0))
		denom := new(big.Int).Sub(fx1, fx0)
		frac, _ := new(big.Float).Quo(new(big.Float).SetInt(nom), new(big.Float).SetInt(denom)).Int(nil)

		x2 := new(big.Int).Sub(x1, frac)
		x0, x1 = x1, x2
		fx0 = fx1

		loopCount += 1
	}

	return nil, ErrNotConverge
}
