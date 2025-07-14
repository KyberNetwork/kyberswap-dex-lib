package poolparty

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

var (
	ErrPoolNotAvailable      = errors.New("pool is currently not available")
	ErrInsufficientLiquidity = errors.New("amount exceeds available tokens")
	ErrInvalidRateToETH      = errors.New("invalid rate to ETH")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	info := pool.PoolInfo{
		Address:     strings.ToLower(entityPool.Address),
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
		Reserves:    []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]), bignumber.NewBig10(entityPool.Reserves[1])},
		BlockNumber: entityPool.BlockNumber,
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool:  pool.Pool{Info: info},
		Extra: extra,
	}, nil
}

// Can only support sell ETH, buy target token
func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == s.Info.Tokens[0] {
		return []string{s.Info.Tokens[1]}
	}

	return nil
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Info.Tokens[1] {
		return []string{s.Info.Tokens[0]}
	}

	return nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !s.Extra.IsVisible || s.Extra.PoolStatus != poolStatusActive {
		return nil, ErrPoolNotAvailable
	}

	if param.TokenAmountIn.Amount.Cmp(s.Extra.PublicAmountAvailable) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	if s.Extra.RateToETH.Sign() == 0 {
		return nil, ErrInvalidRateToETH
	}

	amountOut := new(big.Int).Set(param.TokenAmountIn.Amount)
	amountOut.Mul(amountOut, bignumber.BONE)
	amountOut.Div(amountOut, s.Extra.RateToETH)

	var boostPrice big.Int
	boostPrice.Mul(amountOut, big.NewInt(int64(s.Extra.BoostPriceBps)))
	boostPrice.Div(&boostPrice, bignumber.BasisPoint)

	amountOut.Add(amountOut, &boostPrice)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  s.Info.Tokens[1],
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  s.Info.Tokens[0],
			Amount: nil,
		},
		Gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	s.Extra.PublicAmountAvailable = new(big.Int).Sub(s.Extra.PublicAmountAvailable, params.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{Exchange: s.Extra.Exchange}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator { return s }
