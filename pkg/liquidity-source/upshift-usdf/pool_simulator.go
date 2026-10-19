package upshiftusdf

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator models Vault.deposit(amount): compute() rolls the day and price from the
// clock, then USDF = amount*1e6/tokenPrice. Tokens are [USDC, USDF]; deposit only.
type PoolSimulator struct {
	pool.Pool
	Extra
	// now is the quote clock; tests inject a fixed one.
	now func() time.Time
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.Apr == nil || extra.TokenPrice == nil || extra.MinDeposit == nil || extra.MaxSupply == nil ||
		extra.TotalSupply == nil || len(entityPool.Tokens) != 2 {
		return nil, ErrInvalidExtra
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, _ int) *big.Int { return bignumber.NewBig10(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Extra: extra,
		now:   time.Now,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := param.TokenAmountIn.Token, param.TokenOut
	if tokenIn != s.Info.Tokens[0] || tokenOut != s.Info.Tokens[1] {
		return nil, ErrInvalidSwapPath
	}
	if param.TokenAmountIn.Amount == nil || param.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amount, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	if amount.Lt(s.MinDeposit) {
		return nil, ErrBelowMinDeposit
	}

	// The roll must be the same for any block timestamp the tx can land on.
	ts := uint64(s.now().Unix())
	days, err := rollDays(s.StartOfYear, s.CurrentPeriod, ts)
	if err != nil {
		return nil, err
	}
	for _, t := range [2]uint64{ts + boundaryAfterSec, ts - boundaryBeforeSec} {
		if d, err := rollDays(s.StartOfYear, s.CurrentPeriod, t); err != nil || d != days {
			return nil, ErrPeriodBoundary
		}
	}

	var newPrice uint256.Int
	if err := rolledPrice(&newPrice, s.Apr, s.TokenPrice, days); err != nil {
		return nil, err
	}
	var shares uint256.Int
	if err := mintShares(&shares, amount, &newPrice, s.TotalSupply, s.MaxSupply); err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: shares.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI},
		Gas:            gas(days),
		SwapInfo:       SwapInfo{Days: days, NewPrice: &newPrice, Shares: &shares},
	}, nil
}

// UpdateBalance applies the roll and the mint, as deposit() does on-chain.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok || info.NewPrice == nil || info.Shares == nil {
		return
	}
	// Copy-on-write so clones sharing the old pointers are unaffected.
	s.CurrentPeriod += info.Days
	s.TokenPrice = new(uint256.Int).Set(info.NewPrice)
	s.TotalSupply = new(uint256.Int).Add(s.TotalSupply, info.Shares)
	s.Info.Reserves = Reserves(&s.Extra)
}

// CloneState is shallow: UpdateBalance replaces fields wholesale and never writes through pointers.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{BlockNumber: s.Info.BlockNumber}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Info.Tokens[1] {
		return []string{s.Info.Tokens[0]}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == s.Info.Tokens[0] {
		return []string{s.Info.Tokens[1]}
	}
	return []string{}
}

// Reserves reports the remaining USDF mintable under maxSupply and its USDC value at the
// stored price. maxSupply 0 (unlimited) reports 2^128.
func Reserves(e *Extra) []*big.Int {
	var usdf, usdc uint256.Int
	switch {
	case e.MaxSupply.IsZero():
		usdf.Lsh(uint256.NewInt(1), 128)
	case e.MaxSupply.Gt(e.TotalSupply):
		usdf.Sub(e.MaxSupply, e.TotalSupply)
	}
	if !e.TokenPrice.IsZero() {
		big256.MulDivDown(&usdc, &usdf, e.TokenPrice, usdfMultiplier)
	}
	return []*big.Int{usdc.ToBig(), usdf.ToBig()}
}

func gas(days uint64) int64 {
	if days == 0 {
		return gasNoRoll
	}
	return gasRollBase + int64(days)*gasPerDay
}
