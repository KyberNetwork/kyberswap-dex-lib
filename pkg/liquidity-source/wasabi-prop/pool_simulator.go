package wasabiprop

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
	remainingIn [2]*big.Int
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeWasabiProp)
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
		Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: p.BlockNumber,
		},
		Extra:       extra,
		StaticExtra: staticExtra,
	}

	// Runtime cumulative cap is sample-based for wasabi (no on-chain MaxIn metadata).
	for dir := range 2 {
		if dir < len(sim.Samples) && len(sim.Samples[dir]) > 0 {
			lastIn := sim.Samples[dir][len(sim.Samples[dir])-1][0]
			if lastIn != nil {
				sim.remainingIn[dir] = new(big.Int).Set(lastIn)
			}
		}
	}

	return sim, nil
}

// CalcAmtOut quotes in uint256 amounts. The quote runs in big.Int, converted once at each end.
func (s *PoolSimulator) CalcAmtOut(params pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	tokenOut := strings.ToLower(params.TokenOut)
	amountOutBig, err := s.quote(params.TokenAmountIn.Token, tokenOut, params.TokenAmountIn.Amount.ToBig())
	if err != nil {
		return nil, err
	}
	amountOut, overflow := uint256.FromBig(amountOutBig)
	if overflow {
		return nil, ErrOverflow
	}

	if limit := params.Limit; limit != nil {
		inventoryLimit := limit.GetLimit(tokenOut)
		if inventoryLimit != nil && amountOut.Gt(inventoryLimit) {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: tokenOut, Amount: *amountOut},
		Gas:            defaultGas,
	}, nil
}

// quote returns the amount out for amountIn before any inventory limit. tokenIn and tokenOut are lowercase.
func (s *PoolSimulator) quote(tokenIn, tokenOut string, amountIn *big.Int) (*big.Int, error) {
	tokenIn = strings.ToLower(tokenIn)
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || len(s.Info.Tokens) != 2 {
		return nil, ErrInvalidToken
	}

	if len(s.Samples[indexIn]) == 0 {
		return nil, ErrInsufficientLiquidity
	}
	if rem := s.remainingIn[indexIn]; rem != nil {
		if amountIn.Cmp(rem) > 0 {
			return nil, ErrInsufficientLiquidity
		}
	}

	samples := s.Samples[indexIn]
	idx := sort.Search(len(samples), func(i int) bool {
		return samples[i][0].Cmp(amountIn) > 0
	})
	sampleIndex := max(idx-1, 0)

	var amountOut = new(big.Int)
	bignumber.MulDivDown(amountOut, amountIn, samples[sampleIndex][1], samples[sampleIndex][0])
	return amountOut, nil
}

func (s *PoolSimulator) UpdateState(params pool.UpdateStateParams) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	amountIn := params.TokenAmountIn.Amount.ToBig()
	amountOut := params.TokenAmountOut.Amount.ToBig()
	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], amountIn)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], amountOut)
	if s.remainingIn[indexIn] != nil {
		s.remainingIn[indexIn] = new(big.Int).Sub(s.remainingIn[indexIn], amountIn)
		if s.remainingIn[indexIn].Sign() < 0 {
			s.remainingIn[indexIn] = big.NewInt(0)
		}
	}

	if limit := params.SwapLimit; limit != nil {
		_, _, _ = limit.UpdateLimit(tokenOut, tokenIn, &params.TokenAmountOut.Amount, &params.TokenAmountIn.Amount)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSim {
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
			cloned.remainingIn[i] = new(big.Int).Set(s.remainingIn[i])
		}
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.RouterAddress, BlockNumber: s.Info.BlockNumber}
}

func (s *PoolSimulator) GetReserves() []uint256.Int {
	out := make([]uint256.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		out[i] = *reserveToU256(r)
	}
	return out
}

func (s *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	tokens, reserves := s.GetTokens(), s.GetReserves()
	inventory := make(map[string]*uint256.Int, len(tokens))
	for i, token := range tokens {
		inventory[token] = &reserves[i]
	}
	return inventory
}

// reserveToU256 converts a reserve. A nil, negative, or oversized reserve becomes zero.
func reserveToU256(b *big.Int) *uint256.Int {
	if b == nil || b.Sign() < 0 {
		return new(uint256.Int)
	}
	u, overflow := uint256.FromBig(b)
	if overflow {
		return new(uint256.Int)
	}
	return u
}
