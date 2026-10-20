package pancakeprop

import (
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolSimulator models one PropAMM pair. Token 0 is the lower address, as in the pricing module.
// Not modelled: same-block opposing flow, which only tightens the on-chain price (the simulator
// quotes conservatively), and exact-output swaps.
type PoolSimulator struct {
	pool.Pool
	Extra       Extra
	StaticExtra StaticExtra
	StaleCheck  bool
}

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangePancakeProp)
)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	ep := params.EntityPool
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidState
	}
	s := &PoolSimulator{StaleCheck: params.Opts.StaleCheck}
	if err := json.Unmarshal([]byte(ep.Extra), &s.Extra); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(ep.StaticExtra), &s.StaticExtra); err != nil {
		return nil, err
	}
	// The pricing module stores capacities and cumulatives as uint128.
	for d := range 2 {
		if s.Extra.MaxInput[d].BitLen() > 128 || s.Extra.Cum[d].BitLen() > 128 {
			return nil, ErrInvalidState
		}
	}
	s.Info = pool.PoolInfo{
		Address:     strings.ToLower(ep.Address),
		Exchange:    ep.Exchange,
		Type:        ep.Type,
		Tokens:      []string{ep.Tokens[0].Address, ep.Tokens[1].Address},
		Reserves:    make([]*big.Int, 2),
		BlockNumber: ep.BlockNumber,
	}
	for i, r := range ep.Reserves {
		s.Info.Reserves[i] = bignumber.NewBig(r)
	}
	return s, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if in < 0 || out < 0 || in == out {
		return nil, ErrInvalidToken
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.IsZero() {
		return nil, ErrZeroAmount
	}
	if s.Extra.Paused {
		return nil, ErrPaused
	}
	if s.StaleCheck && uint64(time.Now().UnixMilli()) > s.Extra.ExpiredAtMs {
		return nil, ErrStalePrice
	}

	amountOut, err := unpackPrice(&s.Extra.Packed).quote(in == 0, &s.Extra.MaxInput[in], &s.Extra.Cum[in], amountIn)
	if err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ErrZeroOutput
	}
	// Output is paid from the vault, so its balance caps the swap.
	amountOutBI := amountOut.ToBig()
	if amountOutBI.Cmp(s.Info.Reserves[out]) > 0 {
		return nil, pool.ErrNotEnoughInventory
	}
	if params.Limit != nil {
		if limit := params.Limit.GetLimit(s.limitKey(s.Info.Tokens[out])); limit == nil || amountOutBI.Cmp(limit) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[out], Amount: amountOutBI},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[in], Amount: new(big.Int)},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{IndexIn: in, AmountIn: *amountIn, AmountOut: amountOut},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swap, ok := params.SwapInfo.(SwapInfo)
	if !ok || swap.IndexIn < 0 || swap.IndexIn > 1 {
		return
	}
	in, out := swap.IndexIn, 1-swap.IndexIn
	amountIn, amountOut := swap.AmountIn.ToBig(), swap.AmountOut.ToBig()
	if params.SwapLimit != nil {
		if _, _, err := params.SwapLimit.UpdateLimit(
			s.limitKey(s.Info.Tokens[out]), s.limitKey(s.Info.Tokens[in]), amountOut, amountIn); err != nil {
			return
		}
	}
	s.Info.Reserves[in] = new(big.Int).Add(s.Info.Reserves[in], amountIn)
	s.Info.Reserves[out] = new(big.Int).Sub(s.Info.Reserves[out], amountOut)
	// quote bounded cum + amountIn by the uint128 capacity, so this cannot overflow.
	s.Extra.Cum[in].Add(&s.Extra.Cum[in], &swap.AmountIn)
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Tokens = slices.Clone(s.Info.Tokens)
	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}
	return &cloned
}

// limitKey scopes the inventory to the vault, which backs every pair of the entry.
func (s *PoolSimulator) limitKey(token string) string {
	return s.StaticExtra.Vault + ":" + token
}

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	return map[string]*big.Int{
		s.limitKey(s.Info.Tokens[0]): new(big.Int).Set(s.Info.Reserves[0]),
		s.limitKey(s.Info.Tokens[1]): new(big.Int).Set(s.Info.Reserves[1]),
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{BlockNumber: s.Info.BlockNumber, Entry: s.StaticExtra.Entry}
}
