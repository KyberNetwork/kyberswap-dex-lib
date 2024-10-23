package ondo_usdy

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ondo-usdy/common"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		paused      bool
		oraclePrice *uint256.Int

		gas Gas
	}

	Gas struct {
		Wrap   int64
		Unwrap int64
	}

	SwapInfo struct {
		IsWrap bool `json:"isWrap"`
	}
)

var (
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
		oraclePrice: extra.OraclePrice,
		gas:         defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {

	}

	tokenAmountIn := params.TokenAmountIn
	tokenOut := params.TokenOut

	var tokenInIndex = s.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = s.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &poolpkg.CalcAmountOutResult{}, fmt.Errorf("invalid tokenIn or tokenOut: %v, %v", tokenAmountIn.Token, tokenOut)
	}

	var (
		amountOut *big.Int
		gas       int64
		isWrap    = tokenAmountIn.Token == s.Pool.Info.Tokens[0]
	)

	if isWrap {
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
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gas,
		SwapInfo: SwapInfo{
			IsWrap: isWrap,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(_ poolpkg.UpdateBalanceParams) {}

func (s *PoolSimulator) wrap(USDYAmount *uint256.Int) *big.Int {
	return s.getRUSDYByShares(USDYAmount.Mul(USDYAmount, common.BasisPoints)).ToBig()
}

func (s *PoolSimulator) getRUSDYByShares(shares *uint256.Int) *uint256.Int {
	return shares.
		Mul(shares, s.oraclePrice).
		Div(shares, new(uint256.Int).Mul(number.Number_1e18, common.BasisPoints))
}

func (s *PoolSimulator) unwrap(rUSDYAmount *uint256.Int) (*big.Int, error) {
	usdyAmount := s.getSharesByRUSDY(rUSDYAmount)
	if usdyAmount.Cmp(common.BasisPoints) < 0 {
		return nil, ErrUnwrapTooSmall
	}

	return usdyAmount.Div(usdyAmount, common.BasisPoints).ToBig(), nil
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
