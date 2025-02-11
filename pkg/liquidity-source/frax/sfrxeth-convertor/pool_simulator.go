package sfrxeth_convertor

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	totalSupply *uint256.Int
	totalAssets *uint256.Int

	gas Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		tokens   = make([]string, len(entityPool.Tokens))
		reserves = make([]*big.Int, len(entityPool.Tokens))
	)

	for i := 0; i < len(entityPool.Tokens); i++ {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		totalAssets: uint256.MustFromDecimal(entityPool.Reserves[0]),
		totalSupply: uint256.MustFromDecimal(entityPool.Reserves[1]),
		gas:         defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		amountOut *uint256.Int
		gas       int64
		err       error
	)

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	isDeposit, err := s.isDeposit(params.TokenAmountIn.Token, params.TokenOut)
	if err != nil {
		return nil, err
	}

	if isDeposit {
		amountOut, err = s.deposit(amountIn)
		gas = s.gas.Deposit
	} else {
		// safe check
		if amountIn.Gt(s.totalSupply) {
			return nil, number.ErrUnderflow
		}

		amountOut, err = s.redeem(amountIn)
		gas = s.gas.Redeem
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gas,
		SwapInfo:       SwapInfo{IsDeposit: isDeposit},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amountIn, overflowIn := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, overflowOut := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflowOut || overflowIn {
		return
	}

	isDeposit := params.SwapInfo.(SwapInfo).IsDeposit
	if isDeposit {
		s.totalAssets.Add(s.totalAssets, amountIn)  // deposit frxETH
		s.totalSupply.Add(s.totalSupply, amountOut) // mint sfrxETH
	} else {
		s.totalAssets.Sub(s.totalAssets, amountOut) // withdraw/redeem frxETH
		s.totalSupply.Sub(s.totalSupply, amountIn)  // burn sfrxETH
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) isDeposit(tokenIn, tokenOut string) (bool, error) {
	if tokenIn == s.Pool.Info.Tokens[0] && tokenOut == s.Pool.Info.Tokens[1] {
		return true, nil // frxETH -> sfrxETH
	} else if tokenIn == s.Pool.Info.Tokens[1] && tokenOut == s.Pool.Info.Tokens[0] {
		return false, nil // sfrxETH -> frxETH
	}
	return false, ErrInvalidSwap
}

func (s *PoolSimulator) deposit(assets *uint256.Int) (*uint256.Int, error) {
	if s.totalSupply.IsZero() {
		return assets.Clone(), nil
	}

	// previewDeposit
	shares, overflow := new(uint256.Int).MulDivOverflow(assets, s.totalSupply, s.totalAssets)
	if overflow {
		return nil, number.ErrOverflow
	}

	if shares.IsZero() {
		return nil, ErrZeroDeposit
	}

	return shares, nil
}

func (s *PoolSimulator) redeem(shares *uint256.Int) (*uint256.Int, error) {
	if s.totalSupply.IsZero() {
		return shares.Clone(), nil
	}

	// previewRedeem
	assets, overflow := new(uint256.Int).MulDivOverflow(shares, s.totalAssets, s.totalSupply)
	if overflow {
		return nil, number.ErrOverflow
	}

	if assets.IsZero() {
		return nil, ErrZeroAssets
	}

	if assets.Gt(s.totalAssets) {
		return nil, number.ErrUnderflow
	}

	return assets, nil
}
