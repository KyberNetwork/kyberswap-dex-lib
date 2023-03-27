package makerpsm

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
)

type Pool struct {
	poolpkg.Pool

	PSM PSM

	gas Gas
}

type Extra struct {
	PSM PSM `json:"psm"`
}

type Gas struct {
	BuyGem  int64
	SellGem int64
}

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	var gemDecimals uint8
	for _, poolToken := range entityPool.Tokens {
		if !strings.EqualFold(poolToken.Address, DAIAddress) {
			gemDecimals = poolToken.Decimals
		}

		tokens = append(tokens, poolToken.Address)
	}

	poolInfo := poolpkg.PoolInfo{
		Address:  entityPool.Address,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
	}

	psm := extra.PSM
	psm.To18ConversionFactor = new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(int64(18-gemDecimals)),
		nil,
	)

	return &Pool{
		Pool: poolpkg.Pool{
			Info: poolInfo,
		},
		PSM: psm,
		gas: DefaultGas,
	}, nil
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	if strings.EqualFold(tokenAmountIn.Token, DAIAddress) {
		daiAmt, fee, err := p.PSM.buyGem(tokenAmountIn.Amount)
		if err != nil {
			return &poolpkg.CalcAmountOutResult{}, err
		}
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: daiAmt,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: fee,
			},
			Gas: p.gas.BuyGem,
		}, nil

	}

	gemAmt, fee, err := p.PSM.sellGem(tokenAmountIn.Amount)
	if err != nil {
		return &poolpkg.CalcAmountOutResult{}, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: gemAmt,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: fee,
		},
		Gas: p.gas.SellGem,
	}, nil
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	if strings.EqualFold(input.Token, DAIAddress) {
		p.PSM.updateBalanceBuyingGem(input.Amount)
	}

	p.PSM.updateBalanceSellingGem(output.Amount)
}

func (p *Pool) CanSwapTo(address string) []string {
	isTokenExists := false
	for _, token := range p.Info.Tokens {
		if strings.EqualFold(token, address) {
			isTokenExists = true
		}
	}

	if !isTokenExists {
		return nil
	}

	swappableTokens := make([]string, 0, len(p.Info.Tokens)-1)
	for _, token := range p.Info.Tokens {
		if !strings.EqualFold(token, address) {
			swappableTokens = append(swappableTokens, token)
		}
	}

	return swappableTokens
}

func (p *Pool) GetLpToken() string { return "" }

func (p *Pool) GetMidPrice(tokenIn string, _ string, base *big.Int) *big.Int {
	if strings.EqualFold(tokenIn, DAIAddress) {
		return new(big.Int).Div(new(big.Int).Mul(base, WAD), p.PSM.To18ConversionFactor)
	}

	return new(big.Int).Div(new(big.Int).Mul(base, p.PSM.To18ConversionFactor), WAD)
}

func (p *Pool) CalcExactQuote(tokenIn string, _ string, base *big.Int) *big.Int {
	if strings.EqualFold(tokenIn, DAIAddress) {
		return new(big.Int).Div(new(big.Int).Mul(base, WAD), p.PSM.To18ConversionFactor)
	}

	return new(big.Int).Div(new(big.Int).Mul(base, p.PSM.To18ConversionFactor), WAD)
}

func (p *Pool) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
