package sfrxeth

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
		submitPaused: extra.SubmitPaused,
		totalSupply:  extra.TotalSupply,
		totalAssets:  extra.TotalAssets,
		gas:          defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.submitPaused {
		return nil, ErrSubmitPaused
	}

	amountOut, err := s.submitAndDeposit(uint256.MustFromBig(params.TokenAmountIn.Amount))
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.SubmitAndDeposit,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Pool.Info.Tokens[1] {
		return []string{s.Pool.Info.Tokens[0]}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == s.Pool.Info.Tokens[0] {
		return []string{s.Pool.Info.Tokens[1]}
	}
	return []string{}
}

func (s *PoolSimulator) submitAndDeposit(amountIn *uint256.Int) (*uint256.Int, error) {
	return s.deposit(amountIn)
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
