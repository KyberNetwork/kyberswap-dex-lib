package vaultT1

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	StaticExtra
	Ratio *big.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
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
			SwapFee:     big.NewInt(0), // no swap fee on liquidations
		}},
		StaticExtra: staticExtra,
		Ratio:       extra.Ratio,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmountIn
	}

	tokenAmountOut := new(big.Int).Mul(param.TokenAmountIn.Amount, s.Ratio)

	// ratio is scaled in 1e27, so divide by 1e27
	divisor1e27 := new(big.Int)
	divisor1e27.SetString(String1e27, 10) // 1e27

	tokenAmountOut = new(big.Int).Div(tokenAmountOut, divisor1e27)

	reserveTokenOut, err := s.getReserveForToken(param.TokenOut)
	if err != nil {
		return nil, err
	}

	if tokenAmountOut.Cmp(reserveTokenOut) > 0 {
		return nil, ErrInsufficientReserve
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: tokenAmountOut},
		Fee:            &pool.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            defaultGas.Liquidate,
		SwapInfo:       s.StaticExtra,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount

	for i := range s.Info.Tokens {
		if s.Info.Tokens[i] == input.Token {
			s.Info.Reserves[i] = new(big.Int).Add(s.Info.Reserves[i], inputAmount)
		}
		if s.Info.Tokens[i] == output.Token {
			s.Info.Reserves[i] = new(big.Int).Sub(s.Info.Reserves[i], outputAmount)
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return PoolMeta{
		BlockNumber:     s.Pool.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(valueobject.IsNative(tokenIn), "", s.GetAddress())
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	return s.CanSwapTo(address)
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	if strings.EqualFold(address, s.Info.Tokens[1]) {
		return []string{}
	}

	result := make([]string, 0, len(s.Info.Tokens))
	var tokenIndex = s.GetTokenIndex(address)
	for i := 0; i < len(s.Info.Tokens); i++ {
		if i != tokenIndex {
			result = append(result, s.Info.Tokens[i])
		}
	}

	return result
}

// Helper function to get reserve for a specific token
func (s *PoolSimulator) getReserveForToken(token string) (*big.Int, error) {
	if idx := s.GetTokenIndex(token); idx >= 0 {
		return s.GetReserves()[idx], nil
	}

	return nil, ErrTokenNotFound
}
