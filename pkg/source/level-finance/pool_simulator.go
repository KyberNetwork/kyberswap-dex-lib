package levelfinance

import (
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"math/big"
)

type PoolSimulator struct {
	pool.Pool
	state *PoolState
	gas   Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, len(entityPool.Tokens))
	for i, token := range entityPool.Tokens {
		tokens[i] = token.Address
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Checked:  false,
			},
		},
		state: &PoolState{
			TokenInfos: extra.TokenInfos,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	newState := p.deepCopyState(p.state)
	amountOutAfterFee, err := swap(tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount, newState)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOutAfterFee,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: nil,
		},
		Gas: p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	return
}

func (p *PoolSimulator) deepCopyState(state *PoolState) *PoolState {
	newState := &PoolState{
		TotalWeight:             new(big.Int).Set(state.TotalWeight),
		VirtualPoolValue:        new(big.Int).Set(state.VirtualPoolValue),
		StableCoinBaseSwapFee:   new(big.Int).Set(state.StableCoinBaseSwapFee),
		StableCoinTaxBasisPoint: new(big.Int).Set(state.StableCoinTaxBasisPoint),
		BaseSwapFee:             new(big.Int).Set(state.BaseSwapFee),
		TaxBasisPoint:           new(big.Int).Set(state.TaxBasisPoint),
		DaoFee:                  new(big.Int).Set(state.DaoFee),
	}
	for key, value := range state.TokenInfos {
		newState.TokenInfos[key] = &TokenInfo{
			IsStableCoin:    value.IsStableCoin,
			TargetWeight:    new(big.Int).Set(value.TargetWeight),
			TotalRiskFactor: new(big.Int).Set(value.TotalRiskFactor),
			MinPrice:        new(big.Int).Set(value.MinPrice),
			MaxPrice:        new(big.Int).Set(value.MaxPrice),
		}
		for keyTrancheAsset, valueTrancheAsset := range value.TrancheAssets {
			newState.TokenInfos[key].TrancheAssets[keyTrancheAsset] = &AssetInfo{
				PoolAmount:    new(big.Int).Set(valueTrancheAsset.PoolAmount),
				ReserveAmount: new(big.Int).Set(valueTrancheAsset.ReserveAmount),
			}
		}
		for keyRiskFactor, valueRiskFactor := range value.RiskFactor {
			newState.TokenInfos[key].RiskFactor[keyRiskFactor] = new(big.Int).Set(valueRiskFactor)
		}
	}

	return newState
}
