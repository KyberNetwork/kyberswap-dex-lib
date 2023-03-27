package pool

import (
	"math/big"
)

type IPool interface {
	// CalcAmountOut amountOut, fee, gas
	CalcAmountOut(
		tokenAmountIn TokenAmount,
		tokenOut string,
	) (*CalcAmountOutResult, error)
	UpdateBalance(params UpdateBalanceParams)
	CanSwapTo(address string) []string
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

type CalcAmountOutResult struct {
	TokenAmountOut *TokenAmount
	Fee            *TokenAmount
	Gas            int64
	SwapInfo       interface{}
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
