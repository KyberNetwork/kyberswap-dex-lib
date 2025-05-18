package meth

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mantle/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	isStakingPaused        bool
	minimumStakeBound      *uint256.Int
	maximumMETHSupply      *uint256.Int
	totalControlled        *uint256.Int
	exchangeAdjustmentRate uint16
	mETHTotalSupply        *uint256.Int

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
		isStakingPaused:        extra.IsStakingPaused,
		minimumStakeBound:      extra.MinimumStakeBound,
		maximumMETHSupply:      extra.MaximumMETHSupply,
		totalControlled:        extra.TotalControlled,
		exchangeAdjustmentRate: extra.ExchangeAdjustmentRate,
		mETHTotalSupply:        extra.METHTotalSupply,
		gas:                    defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
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

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Stake,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
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
