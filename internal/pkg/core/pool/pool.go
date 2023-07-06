package pool

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var ErrCalcAmountOutPanic = errors.New("calcAmountOut was panic")

type IPool interface {
	// CalcAmountOut amountOut, fee, gas
	// DO NOT FUCKING MODIFY THE POOL whilst calculating amount out. Call UpdateBalance when you need to.
	CalcAmountOut(
		tokenAmountIn TokenAmount,
		tokenOut string,
	) (*CalcAmountOutResult, error)
	UpdateBalance(params UpdateBalanceParams)
	CanSwapTo(address string) []string
	CanSwapFrom(address string) []string
	GetTokens() []string
	GetAddress() string
	GetExchange() string
	GetType() string
	GetTokenIndex(address string) int
	GetLpToken() string
	GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int
	CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int
	GetMetaInfo(tokenIn string, tokenOut string) interface{}
	Equals(other IPool) bool
}

type Pool struct {
	Info PoolInfo
}

// wrap around pool.CalcAmountOut and catch panic
func CalcAmountOut(pool IPool, tokenAmountIn TokenAmount, tokenOut string) (res *CalcAmountOutResult, err error) {
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

	return pool.CalcAmountOut(tokenAmountIn, tokenOut)
}

func (t *Pool) GetInfo() PoolInfo {
	return t.Info
}

func (t *Pool) GetTokens() []string {
	return t.Info.Tokens
}

func (t *Pool) GetAddress() string {
	return t.Info.Address
}

func (t *Pool) GetExchange() string {
	return t.Info.Exchange
}

func (t *Pool) Equals(other IPool) bool {
	return t.GetAddress() == other.GetAddress()
}

func (t *Pool) GetTokenIndex(address string) int {
	return t.Info.GetTokenIndex(address)
}

func (t *Pool) GetType() string {
	return t.Info.Type
}

// CanSwapTo is the base method to get all swappable tokens from a pool by a given token address
// Pools with custom logic should override this method
func (t *Pool) CanSwapTo(address string) []string {
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 {
		return nil // returning nil is good enough
	}

	result := make([]string, 0, len(t.Info.Tokens)-1) // avoid allocating new memory as much as possible
	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			result = append(result, t.Info.Tokens[i])
		}
	}

	return result
}

// most pools are bi-directional so just call CanSwapTo here
// Pools with custom logic should override this method
func (t *Pool) CanSwapFrom(address string) []string {
	return t.CanSwapTo(address)
}

type CalcAmountOutResult struct {
	TokenAmountOut *TokenAmount
	Fee            *TokenAmount
	Gas            int64
	SwapInfo       interface{}
}

func (r *CalcAmountOutResult) IsValid() bool {
	return r.TokenAmountOut != nil && r.TokenAmountOut.Amount != nil && r.TokenAmountOut.Amount.Cmp(constant.Zero) > 0
}

type UpdateBalanceParams struct {
	TokenAmountIn  TokenAmount
	TokenAmountOut TokenAmount
	Fee            TokenAmount
	SwapInfo       interface{}
}

type PoolToken struct {
	Token               string
	Balance             *big.Int
	Weight              uint
	PrecisionMultiplier *big.Int
	VReserve            *big.Int
}

type PoolInfo struct {
	Address    string
	ReserveUsd float64
	SwapFee    *big.Int
	Exchange   string
	Type       string
	Tokens     []string
	Reserves   []*big.Int
	Checked    bool
}

func (t *PoolInfo) GetTokenIndex(address string) int {
	for i, poolToken := range t.Tokens {
		if poolToken == address {
			return i
		}
	}
	return -1
}
