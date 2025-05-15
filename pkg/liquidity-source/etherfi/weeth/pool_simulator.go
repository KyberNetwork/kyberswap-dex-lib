package weeth

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidAmountIn = errors.New("invalid amountIn")
)

type PoolSimulator struct {
	pool.Pool

	totalShares      *big.Int
	totalPooledEther *big.Int

	gas Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		totalPooledEther: extra.TotalPooledEther,
		totalShares:      extra.TotalShares,
		gas:              defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmountIn
	}

	if param.TokenAmountIn.Token == s.Info.Tokens[0] {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: s.shareForAmount(param.TokenAmountIn.Amount)},
			Fee:            &pool.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
			Gas:            s.gas.Wrap,
		}, nil
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: s.amountForShare(param.TokenAmountIn.Amount)},
		Fee:            &pool.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Unwrap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) shareForAmount(eETHAmount *big.Int) *big.Int {
	if s.totalPooledEther.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	return new(big.Int).Div(new(big.Int).Mul(eETHAmount, s.totalShares), s.totalPooledEther)
}

func (s *PoolSimulator) amountForShare(weETHAmount *big.Int) *big.Int {
	if s.totalShares.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	return new(big.Int).Div(new(big.Int).Mul(weETHAmount, s.totalPooledEther), s.totalShares)
}
