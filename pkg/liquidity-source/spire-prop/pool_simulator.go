package spireprop

import (
	"math"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Extra       Extra
	StaticExtra StaticExtra
	StaleCheck  bool
}

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeSpireProp)
)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	p := params.EntityPool
	if len(p.Tokens) != 2 || len(p.Reserves) != 2 || p.BlockNumber == 0 {
		return nil, ErrInvalidState
	}
	s := &PoolSimulator{StaleCheck: params.Opts.StaleCheck}
	if err := json.Unmarshal([]byte(p.Extra), &s.Extra); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(p.StaticExtra), &s.StaticExtra); err != nil {
		return nil, err
	}
	for _, a := range []string{s.StaticExtra.Entrypoint, s.StaticExtra.CurveBook, s.StaticExtra.Custodian} {
		if !validAddress(a) {
			return nil, ErrInvalidState
		}
	}
	s.Info = pool.PoolInfo{Address: strings.ToLower(p.Address), Exchange: p.Exchange, Type: p.Type, BlockNumber: p.BlockNumber}
	for i, token := range p.Tokens {
		if token == nil || !validAddress(token.Address) {
			return nil, ErrInvalidState
		}
		value, ok := new(big.Int).SetString(p.Reserves[i], 10)
		if !ok || value.Sign() < 0 || value.BitLen() > 256 {
			return nil, ErrInvalidState
		}
		s.Info.Tokens = append(s.Info.Tokens, strings.ToLower(token.Address))
		s.Info.Reserves = append(s.Info.Reserves, value)
	}
	if s.Info.Tokens[0] == s.Info.Tokens[1] {
		return nil, ErrInvalidState
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	if s.expired(s.Extra.BlockTimestamp) || (s.StaleCheck && s.expired(uint64(time.Now().Unix()))) {
		return nil, ErrExpired
	}
	return s, nil
}

func validAddress(a string) bool {
	return common.IsHexAddress(a) && common.HexToAddress(a) != (common.Address{})
}

func (s *PoolSimulator) validate() error {
	e := &s.Extra
	if e.Mid.IsZero() || e.Mid.BitLen() > 80 || e.QUnit.IsZero() || e.CUnit.IsZero() || e.BlockTimestamp == 0 || e.LastUpdateAt > e.BlockTimestamp {
		return ErrInvalidState
	}
	for _, side := range []*Side{&e.Ask, &e.Bid} {
		if side.SpreadBps <= -10000 || side.DepthBps > 10000 || len(side.Knots) > 36 {
			return ErrInvalidState
		}
		var previous Knot
		for i := range side.Knots {
			k := &side.Knots[i]
			if k.Q.BitLen() > 48 || k.Extra.BitLen() > 48 || k.Q.Cmp(&previous.Q) <= 0 || k.Extra.Lt(&previous.Extra) {
				return ErrInvalidState
			}
			if _, _, err := s.atomic(k); err != nil {
				return err
			}
			previous = *k
		}
		if _, err := s.effectiveMax(side); err != nil {
			return err
		}
	}
	return nil
}

func (s *PoolSimulator) deadline() uint64 {
	until := s.Extra.ValidUntil
	// Solidity adds the uint64 timestamps as uint256, so overflow must not wrap.
	if s.Extra.TTL != 0 && s.Extra.TTL <= math.MaxUint64-s.Extra.LastUpdateAt {
		until = min(until, s.Extra.LastUpdateAt+s.Extra.TTL)
	}
	return until
}

func (s *PoolSimulator) expired(now uint64) bool { return now > s.deadline() }

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	in, out := s.GetTokenIndex(strings.ToLower(params.TokenAmountIn.Token)), s.GetTokenIndex(strings.ToLower(params.TokenOut))
	if in < 0 || out < 0 || in == out {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrAmount
	}
	amount, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrAmount
	}
	if s.StaleCheck && s.expired(uint64(time.Now().Unix())) {
		return nil, ErrExpired
	}
	if s.Extra.FillSeq == math.MaxUint64 {
		return nil, ErrOverflow
	}
	var output, cursor uint256.Int
	var err error
	if in == 0 {
		output, cursor, err = s.sell(amount)
	} else {
		output, cursor, err = s.buy(amount)
	}
	if err != nil {
		return nil, err
	}
	outputBig := output.ToBig()
	if output.IsZero() || outputBig.Cmp(s.Info.Reserves[out]) > 0 {
		return nil, pool.ErrNotEnoughInventory
	}
	if params.Limit != nil {
		limit := params.Limit.GetLimit(s.limitKey(s.Info.Tokens[out]))
		if limit == nil || outputBig.Cmp(limit) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}
	// Input is transferred to custody before payout. Account for ERC20 balance
	// arithmetic as well as the curve's own checked arithmetic.
	if new(big.Int).Add(s.Info.Reserves[in], amount.ToBig()).BitLen() > 256 {
		return nil, ErrOverflow
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[out], Amount: outputBig},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[in], Amount: new(big.Int)},
		Gas:            defaultGas + 5000*int64(max(len(s.Extra.Ask.Knots), len(s.Extra.Bid.Knots))),
		SwapInfo:       SwapInfo{IndexIn: in, FillSeq: s.Extra.FillSeq, Cursor: cursor, AmountIn: *amount, AmountOut: output},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swap, ok := params.SwapInfo.(SwapInfo)
	if !ok || swap.IndexIn < 0 || swap.IndexIn > 1 || swap.FillSeq != s.Extra.FillSeq {
		return
	}
	in, out := swap.IndexIn, 1-swap.IndexIn
	amountIn, amountOut := swap.AmountIn.ToBig(), swap.AmountOut.ToBig()
	if params.SwapLimit != nil {
		if _, _, err := params.SwapLimit.UpdateLimit(s.limitKey(s.Info.Tokens[out]), s.limitKey(s.Info.Tokens[in]), amountOut, amountIn); err != nil {
			return
		}
	}
	s.Info.Reserves[in] = new(big.Int).Add(s.Info.Reserves[in], amountIn)
	s.Info.Reserves[out] = new(big.Int).Sub(s.Info.Reserves[out], amountOut)
	if in == 0 {
		s.Extra.Bid.Filled = swap.Cursor
	} else {
		s.Extra.Ask.Filled = swap.Cursor
	}
	s.Extra.FillSeq++
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Tokens = slices.Clone(s.Info.Tokens)
	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, value := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(value)
	}
	cloned.Extra.Ask.Knots = slices.Clone(s.Extra.Ask.Knots)
	cloned.Extra.Bid.Knots = slices.Clone(s.Extra.Bid.Knots)
	return &cloned
}

func (s *PoolSimulator) limitKey(token string) string {
	return strings.ToLower(s.StaticExtra.Custodian) + ":" + token
}

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	return map[string]*big.Int{s.limitKey(s.Info.Tokens[0]): new(big.Int).Set(s.Info.Reserves[0]), s.limitKey(s.Info.Tokens[1]): new(big.Int).Set(s.Info.Reserves[1])}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{BlockNumber: s.Info.BlockNumber, Entrypoint: s.StaticExtra.Entrypoint, Base: s.Info.Tokens[0], ApprovalAddress: s.StaticExtra.Entrypoint, ValidUntil: s.deadline()}
}

func (s *PoolSimulator) GetApprovalAddress(_, _ string) string { return s.StaticExtra.Entrypoint }
