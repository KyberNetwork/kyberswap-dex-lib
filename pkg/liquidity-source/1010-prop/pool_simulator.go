package prop

import (
	"math/big"
	"slices"
	"sort"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
	remainingIn [2]*uint256.Int
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.Exchange1010Prop)
)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	tokens := lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) string { return strings.ToLower(e.Address) })
	reserves := lo.Map(p.Reserves, func(e string, _ int) *big.Int { return bignumber.NewBig(e) })

	sim := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: p.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
	}

	// cumulative cap: prefer MaxIn, otherwise fallback to sampleMaxIn
	for dir := range 2 {
		var rem *uint256.Int
		if dir < len(extra.MaxIn) && extra.MaxIn[dir] != nil {
			rem = new(uint256.Int)
			if rem.SetFromBig(extra.MaxIn[dir]) {
				rem = nil // overflow: treat as uncapped
			}
		} else if dir < len(extra.Samples) && len(extra.Samples[dir]) > 0 {
			last := extra.Samples[dir][len(extra.Samples[dir])-1][0]
			if last != nil {
				rem = new(uint256.Int)
				if rem.SetFromBig(last) {
					rem = nil // overflow: treat as uncapped
				}
			}
		}
		sim.remainingIn[dir] = rem
	}

	return sim, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := params.TokenAmountIn
	tokenOut := strings.ToLower(params.TokenOut)
	tokenIn := strings.ToLower(tokenAmountIn.Token)

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || len(s.Info.Tokens) != 2 {
		return nil, ErrInvalidToken
	}

	if indexIn >= len(s.Samples) || len(s.Samples[indexIn]) == 0 {
		return nil, ErrInsufficientLiquidity
	}

	var amtIn uint256.Int
	if amtIn.SetFromBig(tokenAmountIn.Amount) {
		return nil, ErrInsufficientLiquidity
	}

	if rem := s.remainingIn[indexIn]; rem != nil && amtIn.Gt(rem) {
		return nil, ErrInsufficientLiquidity
	}

	samples := s.Samples[indexIn]
	idx := sort.Search(len(samples), func(i int) bool {
		var si uint256.Int
		si.SetFromBig(samples[i][0])
		return si.Gt(&amtIn)
	})

	var amountOut uint256.Int
	if idx == 0 {
		var s0in, s0out uint256.Int
		s0in.SetFromBig(samples[0][0])
		s0out.SetFromBig(samples[0][1])
		big256.MulDivDown(&amountOut, &amtIn, &s0out, &s0in)
	} else if idx >= len(samples) {
		last := samples[len(samples)-1]
		var lastIn, lastOut uint256.Int
		lastIn.SetFromBig(last[0])
		lastOut.SetFromBig(last[1])
		big256.MulDivDown(&amountOut, &amtIn, &lastOut, &lastIn)
	} else {
		L, R := samples[idx-1], samples[idx]
		var lIn, lOut, rIn, rOut, span, step, delta uint256.Int
		lIn.SetFromBig(L[0])
		lOut.SetFromBig(L[1])
		rIn.SetFromBig(R[0])
		rOut.SetFromBig(R[1])
		span.Sub(&rIn, &lIn)
		step.Sub(&amtIn, &lIn)
		delta.Sub(&rOut, &lOut)
		big256.MulDivDown(&amountOut, &step, &delta, &span)
		amountOut.Add(&amountOut, &lOut)
	}

	if amountOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	if limit := params.Limit; limit != nil {
		inventoryLimit := limit.GetLimit(tokenOut)
		if amountOut.ToBig().Cmp(inventoryLimit) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: big.NewInt(0)},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	if rem := s.remainingIn[indexIn]; rem != nil {
		var amtIn uint256.Int
		amtIn.SetFromBig(params.TokenAmountIn.Amount)
		if rem.Lt(&amtIn) {
			rem.Clear()
		} else {
			rem.Sub(rem, &amtIn)
		}
	}

	if limit := params.SwapLimit; limit != nil {
		_, _, _ = limit.UpdateLimit(
			params.TokenAmountOut.Token,
			params.TokenAmountIn.Token,
			params.TokenAmountOut.Amount,
			params.TokenAmountIn.Amount,
		)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	cloned.Samples = make([][][2]*big.Int, len(s.Samples))
	for i, dir := range s.Samples {
		cloned.Samples[i] = make([][2]*big.Int, len(dir))
		for j, pair := range dir {
			cloned.Samples[i][j] = [2]*big.Int{
				new(big.Int).Set(pair[0]),
				new(big.Int).Set(pair[1]),
			}
		}
	}
	for i := range s.remainingIn {
		if s.remainingIn[i] != nil {
			cloned.remainingIn[i] = new(uint256.Int).Set(s.remainingIn[i])
		}
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.ApprovalInfo{ApprovalAddress: s.RouterAddress}
}

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := s.GetTokens(), s.GetReserves()
	inventory := make(map[string]*big.Int, len(tokens))
	for i, token := range tokens {
		inventory[token] = reserves[i]
	}
	return inventory
}
