package meth

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mantle/common"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		isStakingPaused        bool
		minimumStakeBound      *uint256.Int
		maximumMETHSupply      *uint256.Int
		totalControlled        *uint256.Int
		exchangeAdjustmentRate uint16
		mETHTotalSupply        *uint256.Int

		gas Gas
	}

	Gas struct {
		Stake int64
	}
)

var (
	ErrStakingPaused                 = errors.New("staking paused")
	ErrorInvalidTokenIn              = errors.New("invalid tokenIn")
	ErrorInvalidTokenOut             = errors.New("invalid tokenOut")
	ErrMinimumStakeBoundNotSatisfied = errors.New("minimum stake bound not satisfied")
	ErrMaximumMETHSupplyExceeded     = errors.New("maximum METH supply exceeded")
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
		isStakingPaused:        extra.IsStakingPaused,
		minimumStakeBound:      extra.MinimumStakeBound,
		maximumMETHSupply:      extra.MaximumMETHSupply,
		totalControlled:        extra.TotalControlled,
		exchangeAdjustmentRate: extra.ExchangeAdjustmentRate,
		mETHTotalSupply:        extra.METHTotalSupply,
		gas:                    defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.isStakingPaused {
		return nil, ErrStakingPaused
	}

	if params.TokenAmountIn.Token != WETH {
		return nil, ErrorInvalidTokenIn
	}

	if params.TokenOut != METH {
		return nil, ErrorInvalidTokenOut
	}

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	if amountIn.Cmp(s.minimumStakeBound) < 0 {
		return nil, ErrMinimumStakeBoundNotSatisfied
	}

	amountOut, err := s.ethToMETH(amountIn)
	if err != nil {
		return nil, err
	}

	if new(uint256.Int).Add(amountOut, s.mETHTotalSupply).Cmp(s.maximumMETHSupply) > 0 {
		return nil, ErrMaximumMETHSupplyExceeded
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Stake,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	s.mETHTotalSupply.Add(s.mETHTotalSupply, uint256.MustFromBig(params.TokenAmountOut.Amount))
}

func (s *PoolSimulator) ethToMETH(mETHAmount *uint256.Int) (*uint256.Int, error) {
	// 1:1 exchange rate on the first stake
	if s.mETHTotalSupply.IsZero() {
		return mETHAmount, nil
	}

	var mETHSupplyAdjusted, totalControlledAdjusted uint256.Int
	mETHSupplyAdjusted.SetUint64(uint64(common.UInt16BasisPoints-s.exchangeAdjustmentRate)).
		Mul(s.mETHTotalSupply, &mETHSupplyAdjusted)

	totalControlledAdjusted.Set(common.BasisPoints).
		Mul(s.totalControlled, &totalControlledAdjusted)

	amountOut, overflow := new(uint256.Int).MulDivOverflow(mETHAmount, &mETHSupplyAdjusted, &totalControlledAdjusted)
	if overflow {
		return nil, number.ErrOverflow
	}

	return amountOut, nil
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == METH {
		return []string{WETH}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == WETH {
		return []string{METH}
	}
	return []string{}
}
