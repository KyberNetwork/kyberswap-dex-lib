package ondo_usdy

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ondo-usdy/common"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		paused      bool
		totalShares *uint256.Int
		oraclePrice *uint256.Int

		gas Gas
	}

	Gas struct {
		Wrap   int64
		Unwrap int64
	}
)

var (
	ErrPoolPaused     = errors.New("pool is paused")
	ErrUnwrapTooSmall = errors.New("unwrap too small")
)

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
		paused:      extra.Paused,
		totalShares: extra.TotalShares,
		oraclePrice: extra.OraclePrice,
		gas:         defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	tokenAmountIn := params.TokenAmountIn
	tokenOut := params.TokenOut

	var tokenInIndex = s.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = s.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &poolpkg.CalcAmountOutResult{}, fmt.Errorf("invalid tokenIn or tokenOut: %v, %v", tokenAmountIn.Token, tokenOut)
	}

	var (
		amountOut *uint256.Int
		gas       int64
	)

	if s.Pool.GetAddress() == tokenOut {
		amountOut = s.wrap(uint256.MustFromBig(tokenAmountIn.Amount))
		gas = s.gas.Wrap
	} else {
		usdySharesAmount, err := s.unwrap(uint256.MustFromBig(tokenAmountIn.Amount))
		if err != nil {
			return nil, err
		}
		amountOut = usdySharesAmount
		gas = s.gas.Unwrap
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	if s.Pool.GetAddress() == params.TokenAmountOut.Token {
		shares := new(uint256.Int).Mul(uint256.MustFromBig(params.TokenAmountIn.Amount), common.BasisPoints)
		s.totalShares.Add(s.totalShares, shares)
	} else {
		shares := s.getSharesByRUSDY(uint256.MustFromBig(params.TokenAmountIn.Amount))
		s.totalShares.Sub(s.totalShares, shares)
	}
}

func (s *PoolSimulator) wrap(USDYAmount *uint256.Int) *uint256.Int {
	shares := new(uint256.Int).Mul(USDYAmount, common.BasisPoints)
	return s.getRUSDYByShares(shares)
}

func (s *PoolSimulator) getRUSDYByShares(shares *uint256.Int) *uint256.Int {
	var temp uint256.Int
	return temp.Mul(shares, s.oraclePrice).
		Div(&temp, new(uint256.Int).Mul(number.Number_1e18, common.BasisPoints))
}

func (s *PoolSimulator) unwrap(rUSDYAmount *uint256.Int) (*uint256.Int, error) {
	usdySharesAmount := s.getSharesByRUSDY(rUSDYAmount)
	if usdySharesAmount.Cmp(common.BasisPoints) < 0 {
		return nil, ErrUnwrapTooSmall
	}

	if usdySharesAmount.Cmp(s.totalShares) > 0 {
		return nil, number.ErrUnderflow
	}

	return usdySharesAmount.Div(usdySharesAmount, common.BasisPoints), nil
}

func (s *PoolSimulator) getSharesByRUSDY(rUSDYAmount *uint256.Int) *uint256.Int {
	return rUSDYAmount.
		Mul(rUSDYAmount, number.Number_1e18).
		Mul(rUSDYAmount, common.BasisPoints).
		Div(rUSDYAmount, s.oraclePrice)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}
