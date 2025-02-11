package makerpsm

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool

	PSM PSM

	gas Gas
}

var _ = pool.RegisterFactory0(DexTypeMakerPSM, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
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

	poolInfo := pool.PoolInfo{
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

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: poolInfo,
		},
		PSM: psm,
		gas: DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	if strings.EqualFold(tokenAmountIn.Token, DAIAddress) {
		daiAmt, fee, err := p.PSM.buyGem(tokenAmountIn.Amount)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
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
		return &pool.CalcAmountOutResult{}, err
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

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	if strings.EqualFold(input.Token, DAIAddress) {
		p.PSM.updateBalanceBuyingGem(input.Amount)
		return
	}

	p.PSM.updateBalanceSellingGem(output.Amount)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
