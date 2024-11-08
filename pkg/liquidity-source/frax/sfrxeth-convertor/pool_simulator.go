package sfrxeth_convertor

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		submitPaused bool
		totalSupply  *uint256.Int
		totalAssets  *uint256.Int

		gas Gas
	}

	SwapInfo struct {
		SwapType uint8 `json:"swapType"`
	}
)

var (
	ErrInvalidSwap     = errors.New("invalid swap")
	ErrSubmitPaused    = errors.New("submit is paused")
	ErrInvalidTokenIn  = errors.New("invalid tokenIn")
	ErrInvalidTokenOut = errors.New("invalid tokenOut")
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
		totalSupply: extra.TotalSupply,
		totalAssets: extra.TotalAssets,
		gas:         defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.submitPaused {
		return nil, ErrSubmitPaused
	}

	tokenAmountIn := params.TokenAmountIn
	tokenOut := params.TokenOut
	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	var (
		amountOut *uint256.Int
		gas       int64
		err       error
	)

	swapType := s.getSwapType(tokenAmountIn.Token, tokenOut)
	switch swapType {
	case Deposit:
		amountOut, err = s.deposit(amountIn)
		gas = s.gas.Deposit
	case Redeem:
		amountOut, err = s.redeem(amountIn)
		gas = s.gas.Redeem
	case InvalidSwap:
		return nil, ErrInvalidSwap
	}

	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gas,
		SwapInfo:       SwapInfo{SwapType: uint8(swapType)},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {

}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) getSwapType(tokenIn, tokenOut string) int {
	if tokenIn == s.Pool.Info.Tokens[0] && tokenOut == s.Pool.Info.Tokens[1] {
		return Deposit // frxETH -> sfrxETH
	} else if tokenIn == s.Pool.Info.Tokens[1] && tokenOut == s.Pool.Info.Tokens[0] {
		return Redeem // sfrxETH -> frxETH
	}
	return InvalidSwap
}

func (s *PoolSimulator) deposit(assets *uint256.Int) (*uint256.Int, error) {
	if s.totalSupply.IsZero() {
		return assets.Clone(), nil
	}

	var (
		shares   uint256.Int
		overflow bool
	)
	_, overflow = shares.MulDivOverflow(assets, s.totalSupply, s.totalAssets)
	if overflow {
		return nil, number.ErrOverflow
	}
	return &shares, nil
}

func (s *PoolSimulator) redeem(shares *uint256.Int) (*uint256.Int, error) {
	if s.totalSupply.IsZero() {
		return shares.Clone(), nil
	}

	var (
		assets   uint256.Int
		overflow bool
	)
	_, overflow = assets.MulDivOverflow(shares, s.totalAssets, s.totalSupply)
	if overflow {
		return nil, number.ErrOverflow
	}
	return &assets, nil
}
