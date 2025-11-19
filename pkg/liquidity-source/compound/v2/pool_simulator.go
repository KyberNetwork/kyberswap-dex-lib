package v2

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	extra Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		extra: extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(param.TokenAmountIn.Token), s.GetTokenIndex(param.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, fmt.Errorf("invalid token")
	}

	isMint := indexIn == 1
	if s.extra.IsMintPaused && isMint {
		return nil, fmt.Errorf("mint is paused")
	}

	var amountOut big.Int

	if isMint {
		// mint: underlying -> cToken
		// amountOut = amountIn / exchangeRate
		amountOut.Mul(param.TokenAmountIn.Amount, bignumber.BONE)
		amountOut.Div(&amountOut, s.extra.ExchangeRateStored)
	} else {
		// redeem: cToken -> underlying
		// amountOut = amountIn * exchangeRate
		amountOut.Mul(param.TokenAmountIn.Amount, s.extra.ExchangeRateStored)
		amountOut.Div(&amountOut, bignumber.BONE)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: &amountOut},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: integer.Zero()},
		Gas:            lo.Ternary(isMint, mintGas, redeemGas),
	}, nil
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator { return s }

func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}
