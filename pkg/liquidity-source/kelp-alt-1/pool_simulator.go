package rsethalt1

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/samber/lo"
)

var (
	ErrInvalidTokenOut        = errors.New("invalid tokenOut")
	ErrInvalidAmountToDeposit = errors.New("invalid amount to deposit")
)

type PoolSimulator struct {
	poolpkg.Pool

	priceByAsset map[string]*big.Int
	feeBps       *big.Int
	gas          Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		priceByAsset: extra.PriceByAsset,
		feeBps:       extra.FeeBps,
		gas:          defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if param.TokenOut != s.Info.Tokens[0] {
		return nil, ErrInvalidTokenOut
	}

	if param.TokenAmountIn.Amount.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrInvalidAmountToDeposit
	}
	isWstETH := false
	if len(s.Info.Tokens) >= 3 && strings.EqualFold(param.TokenAmountIn.Token, s.Info.Tokens[2]) {
		isWstETH = true
	}

	amountOut, err := s._viewSwapRsETHAmountAndFee(param.TokenAmountIn.Amount, isWstETH)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: amountOut},
		Fee:            &poolpkg.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.DepositAsset,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param poolpkg.UpdateBalanceParams) {
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) _viewSwapRsETHAmountAndFee(amount *big.Int, isWstETH bool) (*big.Int, error) {
	fee := new(big.Int).Div(
		new(big.Int).Mul(amount, s.feeBps),
		big.NewInt(10_000),
	)
	amountAfterFee := new(big.Int).Sub(amount, fee)
	if isWstETH {
		// Adjust for wstETH to ETH conversion using the oracle
		ethPrice := s.priceByAsset[s.Info.Tokens[2]]
		amountAfterFee = new(big.Int).Div(
			new(big.Int).Mul(amountAfterFee, ethPrice),
			bignumber.BONE,
		)
	}

	rsETHToETHrate := s.priceByAsset[s.Info.Tokens[0]]
	rsETHAmount := new(big.Int).Div(
		new(big.Int).Mul(amountAfterFee, bignumber.BONE),
		rsETHToETHrate,
	)
	return rsETHAmount, nil
}
