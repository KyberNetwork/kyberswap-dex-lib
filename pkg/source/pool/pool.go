package pool

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/logger"
)

var (
	ErrCalcAmountOutPanic = errors.New("calcAmountOut was panic")
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
	TokenAmountOut *TokenAmount
	Fee            *TokenAmount
	Gas            int64
	SwapInfo       interface{}
}

func (r *CalcAmountOutResult) IsValid() bool {
	return r.TokenAmountOut != nil && r.TokenAmountOut.Amount != nil && r.TokenAmountOut.Amount.Cmp(ZeroBI) > 0
}

type CalcAmountInResult struct {
	TokenAmountIn *TokenAmount
	Fee           *TokenAmount
	Gas           int64
	SwapInfo      interface{}
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

// wrap around pool.CalcAmountOut and catch panic
func CalcAmountOut(pool IPoolSimulator, tokenAmountIn TokenAmount, tokenOut string, limit SwapLimit) (res *CalcAmountOutResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrCalcAmountOutPanic
			logger.WithFields(
				logger.Fields{
					"recover":     r,
					"poolAddress": pool.GetAddress(),
				}).Warn(err.Error())
		}
	}()

	return pool.CalcAmountOut(
		CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
			Limit:         limit,
		})
}

// CalcAmountIn we will run CalcAmountOut twice to find the approximate amountIn
// For example, we need to calculate how many of token X we need to swap to get 1 ETH
// 1st calculation: we will calculate from 1 ETH, how many token X we will get => 1 ETH => k token X
// 2nd calculation: we will calculate from k token X, how many ETH we will get => k token X => 0.9 ETH for example
// After 2 calculations, we have the rate k token X => 0.9 ETH
// To get 1 ETH, we need k/0.9 token X
func CalcAmountIn(pool IPoolSimulator, tokenAmountOut TokenAmount, tokenIn string, limit SwapLimit) (res *CalcAmountInResult, err error) {
	// 1st calculation
	// We calculate from tokenAmountOut of tokenOut, how many tokenIn we can get (let's call this value X)
	amountOutTokenIn, err := pool.CalcAmountOut(CalcAmountOutParams{
		TokenAmountIn: tokenAmountOut,
		TokenOut:      tokenIn,
		Limit:         limit,
	})
	if err != nil {
		return nil, err
	}

	// Now we do the 2nd calculation
	// We will calculate from X tokenIn, how many tokenOut we can get
	amountOutTokenOut, err := pool.CalcAmountOut(
		CalcAmountOutParams{
			TokenAmountIn: *amountOutTokenIn.TokenAmountOut,
			TokenOut:      tokenAmountOut.Token,
			Limit:         limit},
	)
	if err != nil {
		return nil, err
	}

	// Now we calculate the amountIn of tokenIn we need to get tokenAmountOut of tokenOut
	amountIn := new(big.Int).Div(new(big.Int).Mul(tokenAmountOut.Amount, amountOutTokenIn.TokenAmountOut.Amount), amountOutTokenOut.TokenAmountOut.Amount)

	return &CalcAmountInResult{
		TokenAmountIn: &TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		Fee: &TokenAmount{
			Token:  tokenAmountOut.Token,
			Amount: nil,
		},
		Gas: amountOutTokenOut.Gas,
	}, nil
}
