//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PoolSimulator
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package fraxswap

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
)

var FeePrecision = big.NewInt(10000) // basis point, fixed in contract

type (
	PoolSimulator struct {
		pool.Pool

		Fee      *big.Int
		Reserve0 *big.Int
		Reserve1 *big.Int

		gas Gas
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, token := range entityPool.Tokens {
		tokens = append(tokens, token.Address)
	}

	reserves := make([]*big.Int, 0, len(entityPool.Reserves))
	for _, reserve_s := range entityPool.Reserves {
		reserve, ok := new(big.Int).SetString(reserve_s, 10)
		if !ok {
			err := errors.New("failed to parse pool reserve")
			logger.WithFields(logger.Fields{
				"reserve": reserve_s,
				"address": entityPool.Address,
			}).Debug(err.Error())

			return nil, err
		}
		reserves = append(reserves, reserve)
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
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

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOut,
	}

	fee := &pool.TokenAmount{
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

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
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

func (p *PoolSimulator) GetLpToken() string {
	return ""
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
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

func (p *PoolSimulator) GetMidPrice(tokenIn string, _ string, base *big.Int) *big.Int {
	exactQuote, err := p.getAmountOut(base, tokenIn)
	if err != nil {
		return bignumber.ZeroBI
	}

	return exactQuote
}

func (p *PoolSimulator) CalcExactQuote(tokenIn string, _ string, base *big.Int) *big.Int {
	exactQuote, err := p.getAmountOut(base, tokenIn)
	if err != nil {
		return bignumber.ZeroBI
	}

	return exactQuote
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	swapFee := new(big.Int).Sub(FeePrecision, p.Fee)

	return Meta{
		SwapFee:      uint32(swapFee.Uint64()),
		FeePrecision: uint32(FeePrecision.Int64()),
	}
}

// getAmountOut given an input amount of an asset and pair reserves, returns the maximum output amount of the other asset
// amountOut = (amountIn * fee * reserveOut) / ((reserveIn * 10000) + (amountIn * fee))
// https://github.com/FraxFinance/frax-solidity/blob/012909d168ec0eb549aa9689c0d5cd0cafee400b/src/echidna/FraxswapPairV2.sol#L868
func (p *PoolSimulator) getAmountOut(amountIn *big.Int, tokenIn string) (*big.Int, error) {
	var (
		reserveIn  *big.Int
		reserveOut *big.Int
	)

	if strings.EqualFold(tokenIn, p.Info.Tokens[0]) {
		reserveIn, reserveOut = p.Reserve0, p.Reserve1
	} else {
		reserveIn, reserveOut = p.Reserve1, p.Reserve0
	}

	if amountIn.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	if reserveIn.Cmp(bignumber.ZeroBI) <= 0 || reserveOut.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountInWithFee := new(big.Int).Mul(amountIn, p.Fee)
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(big.Int).Add(new(big.Int).Mul(reserveIn, FeePrecision), amountInWithFee)

	return new(big.Int).Div(numerator, denominator), nil
}
