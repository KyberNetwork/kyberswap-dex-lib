package thogprop

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool

	staticExtra StaticExtra
	extra       Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	if len(ep.Tokens) != len(tokenTable) {
		return nil, errors.New("thog-prop: pool must have exactly 8 tokens")
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	if len(ep.Extra) == 0 {
		return nil, errors.New("thog-prop: pool extra is empty")
	}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, len(ep.Tokens))
	for i, t := range ep.Tokens {
		tokens[i] = t.Address
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      tokens,
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bigFromString(s) }),
			BlockNumber: ep.BlockNumber,
		}},
		staticExtra: staticExtra,
		extra:       extra,
	}, nil
}

func bigFromString(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return new(big.Int)
	}
	return v
}

func (s *PoolSimulator) stateFromExtra() (*State, error) {
	parse := func(str string) (*big.Int, error) {
		v, ok := new(big.Int).SetString(str, 10)
		if !ok {
			return nil, errors.New("thog-prop: malformed packed state word")
		}
		return v, nil
	}
	v1, err := parse(s.extra.V1)
	if err != nil {
		return nil, err
	}
	v2, err := parse(s.extra.V2)
	if err != nil {
		return nil, err
	}
	v3, err := parse(s.extra.V3)
	if err != nil {
		return nil, err
	}
	r1, err := parse(s.extra.R1)
	if err != nil {
		return nil, err
	}
	r2, err := parse(s.extra.R2)
	if err != nil {
		return nil, err
	}
	r3, err := parse(s.extra.R3)
	if err != nil {
		return nil, err
	}
	i1, err := parse(s.extra.I1)
	if err != nil {
		return nil, err
	}
	i2, err := parse(s.extra.I2)
	if err != nil {
		return nil, err
	}
	p1, err := parse(s.extra.P1)
	if err != nil {
		return nil, err
	}
	p2, err := parse(s.extra.P2)
	if err != nil {
		return nil, err
	}

	if len(s.Info.Reserves) != len(tokenTable) {
		return nil, errors.New("thog-prop: reserves length mismatch")
	}
	balances := make([]*big.Int, len(tokenTable))
	for i, r := range s.Info.Reserves {
		if r == nil {
			return nil, errors.New("thog-prop: nil reserve")
		}
		balances[i] = new(big.Int).Set(r)
	}

	return &State{
		V1: v1, V2: v2, V3: v3,
		R1: r1, R2: r2, R3: r3,
		I1: i1, I2: i2,
		P1: p1, P2: p2,
		GloballyPaused: s.extra.GloballyPaused,
		RiskV3Ready:    s.extra.RiskV3Ready,
		PairRiskReady:  s.extra.PairRiskReady,
		Balances:       balances,
	}, nil
}

// CalcAmountOut runs ExactQuote (math.go) against the tracker's last polled
// snapshot. fastLaneHot is always forced true here: makerQuoteExactInput()
// itself excludes the AuctionHandler's 9990/10000 haircut, but a real
// aggregator-routed swap may hit it already warm, so quoting the haircut is
// the conservative choice -- see constant.go's fastLaneFrictionBps comment.
// executionBlock adds a small fixed lookahead to the snapshot's own block
// (executionAgeLookahead) rather than assuming age=0, since age directly
// widens the spread and a maxAge violation hard-reverts on-chain.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, ok := tokenByAddress(params.TokenAmountIn.Token)
	if !ok {
		return nil, ErrInvalidToken
	}
	tokenOut, ok := tokenByAddress(params.TokenOut)
	if !ok {
		return nil, ErrInvalidToken
	}

	state, err := s.stateFromExtra()
	if err != nil {
		return nil, err
	}

	amountOut, _, err := ExactQuote(
		state, tokenIn, tokenOut,
		params.TokenAmountIn.Amount,
		s.extra.SnapshotBlock+executionAgeLookahead,
		true,
	)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: new(big.Int)},
		Gas:            defaultGas,
		SwapInfo:       nil,
	}, nil
}

// UpdateBalance adjusts local in-memory reserves so a chained multi-hop
// route within one search sees the depletion of its own prior hop. This is
// a local bookkeeping approximation, not a replay of any on-chain state
// transition -- ThogAMM's real inventory/risk state (i1/i2/exposure) only
// moves via the next tracker poll of makerSnapshot().
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn, ok := tokenByAddress(params.TokenAmountIn.Token)
	if !ok {
		return
	}
	tokenOut, ok := tokenByAddress(params.TokenAmountOut.Token)
	if !ok {
		return
	}

	s.Info.Reserves[tokenIn.Index] = new(big.Int).Add(s.Info.Reserves[tokenIn.Index], params.TokenAmountIn.Amount)
	if s.Info.Reserves[tokenOut.Index].Cmp(params.TokenAmountOut.Amount) >= 0 {
		s.Info.Reserves[tokenOut.Index] = new(big.Int).Sub(s.Info.Reserves[tokenOut.Index], params.TokenAmountOut.Amount)
	} else {
		s.Info.Reserves[tokenOut.Index] = new(big.Int)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int {
		if r == nil {
			return nil
		}
		return new(big.Int).Set(r)
	})
	return &cloned
}

// GetMetaInfo uses the shared pool.MetaInfo shape: ThogAMM's proxy is the
// single address to approve (pull payment, per makerSwapExactInput's
// signature having no explicit fund-transfer step) and to call.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.Info.Address, BlockNumber: s.Info.BlockNumber}
}
