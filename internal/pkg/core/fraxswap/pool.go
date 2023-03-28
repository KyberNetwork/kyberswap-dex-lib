package fraxswap

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
)

var FeePrecision = big.NewInt(10000) // basis point, fixed in contract

type (
	Pool struct {
		poolpkg.Pool

		Fee      *big.Int
		Reserve0 *big.Int
		Reserve1 *big.Int

		gas Gas
	}
	Gas struct {
		Swap int64
	}

	Extra struct {
		// Reserve0 reserve0 after twamm
		Reserve0 *big.Int `json:"reserve0"`

		// Reserve1 reserve1 after twamm
		Reserve1 *big.Int `json:"reserve1"`

		// Fee = 10000 - feeInBasisPoint
		// if fee is 0.3% -> Fee = 9970
		Fee *big.Int `json:"fee"`
	}

	Meta struct {
		SwapFee      uint32 `json:"swapFee"`
		FeePrecision uint32 `json:"feePrecision"`
	}
)

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, token := range entityPool.Tokens {
		tokens = append(tokens, token.Address)
	}

	reserves := make([]*big.Int, 0, len(entityPool.Reserves))
	for _, reserve := range entityPool.Reserves {
		reserves = append(reserves, utils.NewBig10(reserve))
	}

	return &Pool{
		Pool: poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		Fee:      extra.Fee,
		Reserve0: extra.Reserve0,
		Reserve1: extra.Reserve1,
		gas:      DefaultGas,
	}, nil
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var (
		reserveOut *big.Int
	)

	if strings.EqualFold(tokenAmountIn.Token, p.Info.Tokens[0]) {
		reserveOut = p.Reserve1
	} else {
		reserveOut = p.Reserve0
	}

	amountOut, err := p.getAmountOut(tokenAmountIn.Amount, tokenAmountIn.Token)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	if amountOut.Cmp(reserveOut) >= 0 {
		return &pool.CalcAmountOutResult{}, ErrInsufficientLiquidity
	}

	tokenAmountOut := &poolpkg.TokenAmount{
		Token:  tokenOut,
		Amount: amountOut,
	}

	fee := &poolpkg.TokenAmount{
		Token: tokenAmountIn.Token,
		Amount: new(big.Int).Sub(
			tokenAmountIn.Amount,
			new(big.Int).Div(
				new(big.Int).Mul(tokenAmountIn.Amount, p.Fee),
				FeePrecision,
			),
		),
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: tokenAmountOut,
		Fee:            fee,
		Gas:            p.gas.Swap,
	}, nil
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	amountOut, err := p.getAmountOut(params.TokenAmountIn.Amount, params.TokenAmountIn.Token)
	if err != nil {
		return
	}

	amountIn := new(big.Int).Div(
		new(big.Int).Mul(params.TokenAmountIn.Amount, p.Fee),
		FeePrecision,
	)

	if strings.EqualFold(params.TokenAmountIn.Token, p.Info.Tokens[0]) {
		p.Reserve0 = new(big.Int).Add(p.Reserve0, amountIn)
		p.Reserve1 = new(big.Int).Sub(p.Reserve1, amountOut)

		return
	}

	p.Reserve0 = new(big.Int).Sub(p.Reserve0, amountOut)
	p.Reserve1 = new(big.Int).Add(p.Reserve1, amountIn)
}

func (p *Pool) GetLpToken() string {
	return ""
}

func (p *Pool) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = p.GetTokenIndex(address)
	if tokenIndex < 0 {
		return ret
	}
	for i := 0; i < len(p.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, p.Info.Tokens[i])
		}
	}
	return ret
}

func (p *Pool) GetMidPrice(tokenIn string, _ string, base *big.Int) *big.Int {
	exactQuote, err := p.getAmountOut(base, tokenIn)
	if err != nil {
		return constant.Zero
	}

	return exactQuote
}

func (p *Pool) CalcExactQuote(tokenIn string, _ string, base *big.Int) *big.Int {
	exactQuote, err := p.getAmountOut(base, tokenIn)
	if err != nil {
		return constant.Zero
	}

	return exactQuote
}

func (p *Pool) GetMetaInfo(_ string, _ string) interface{} {
	swapFee := new(big.Int).Sub(FeePrecision, p.Fee)

	return Meta{
		SwapFee:      uint32(swapFee.Uint64()),
		FeePrecision: uint32(FeePrecision.Int64()),
	}
}

// getAmountOut given an input amount of an asset and pair reserves, returns the maximum output amount of the other asset
// amountOut = (amountIn * fee * reserveOut) / ((reserveIn * 10000) + (amountIn * fee))
// https://github.com/FraxFinance/frax-solidity/blob/012909d168ec0eb549aa9689c0d5cd0cafee400b/src/echidna/FraxswapPairV2.sol#L868
func (p *Pool) getAmountOut(amountIn *big.Int, tokenIn string) (*big.Int, error) {
	var (
		reserveIn  *big.Int
		reserveOut *big.Int
	)

	if strings.EqualFold(tokenIn, p.Info.Tokens[0]) {
		reserveIn, reserveOut = p.Reserve0, p.Reserve1
	} else {
		reserveIn, reserveOut = p.Reserve1, p.Reserve0
	}

	if amountIn.Cmp(constant.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	if reserveIn.Cmp(constant.Zero) <= 0 || reserveOut.Cmp(constant.Zero) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountInWithFee := new(big.Int).Mul(amountIn, p.Fee)
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(big.Int).Add(new(big.Int).Mul(reserveIn, FeePrecision), amountInWithFee)

	return new(big.Int).Div(numerator, denominator), nil
}
