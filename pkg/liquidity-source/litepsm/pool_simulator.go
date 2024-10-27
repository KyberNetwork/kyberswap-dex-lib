package litepsm

import (
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/bytedance/sonic"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	litePSM LitePSM
	gem     Token
	gas     Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := sonic.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	gem := staticExtra.Gem

	var extra Extra
	if err := sonic.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, poolToken := range entityPool.Tokens {
		tokens = append(tokens, poolToken.Address)
	}

	poolInfo := pool.PoolInfo{
		Address:  entityPool.Address,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
	}

	litePSM := extra.LitePSM
	litePSM.To18ConversionFactor = new(uint256.Int).Exp(
		number.Number_10,
		uint256.NewInt(uint64(18-gem.Decimals)),
	)

	daiBalance, err := uint256.FromDecimal(entityPool.Reserves[0])
	if err != nil {
		return nil, err
	}
	litePSM.DaiBalance = daiBalance

	gemBalance, err := uint256.FromDecimal(entityPool.Reserves[1])
	if err != nil {
		return nil, err
	}
	litePSM.GemBalance = gemBalance

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: poolInfo,
		},
		litePSM: litePSM,
		gem:     gem,
		gas:     DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	if strings.EqualFold(tokenAmountIn.Token, DAIAddress) {
		daiAmt, fee, err := p.litePSM.buyGem(amountIn)
		if err != nil {
			return nil, err
		}

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: daiAmt.ToBig(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: fee.ToBig(),
			},
			Gas: p.gas.BuyGem,
		}, nil
	}

	if strings.EqualFold(tokenAmountIn.Token, p.gem.Address) {
		daiAmt, fee, err := p.litePSM.sellGem(amountIn)
		if err != nil {
			return nil, err
		}

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: daiAmt.ToBig(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: fee.ToBig(),
			},
			Gas: p.gas.SellGem,
		}, nil
	}

	return nil, ErrInvalidToken
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut

	inputAmount, overflowIn := uint256.FromBig(input.Amount)
	outputAmount, overflowOut := uint256.FromBig(output.Amount)
	if overflowIn || overflowOut {
		// This should never happen
		return
	}

	if strings.EqualFold(input.Token, DAIAddress) {
		p.litePSM.updateBalanceBuyingGem(inputAmount, outputAmount)
	}

	p.litePSM.updateBalanceSellingGem(inputAmount, outputAmount)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
