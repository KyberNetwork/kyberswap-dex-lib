package nemoprop

import (
	"math/big"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolSimulator quotes one (base, market) of a NemoSwap proxy from Nemo's
// live feed only: every quote on a base simulator reads the latest snapshot,
// decayed once it is past the fresh window, and with none quotable it
// returns ErrNoFeed. The feed
// connection is looked up by key, never stored here, because simulators
// are msgpack-serialized including unexported fields.
type PoolSimulator struct {
	pool.Pool

	proxy  string
	base   string
	market string

	feed        FeedSettings
	feedKey     string
	feedVersion int64 // pool state timestamp, orders feed settings across pool states

	// pinned is this simulator's own copy of the snapshot state a route is
	// bound to, set by CloneState and UpdateBalance and never shared. Nil on
	// base simulators, which always follow the latest snapshot.
	pinned      *ladder.PoolSimulator
	pinnedSeq   uint64
	pinnedDecay uint64
}

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeNemoProp)
)

// NewPoolSimulator ignores params.Opts.StaleCheck: pool state carries no
// prices to go stale, and every quote checks the live feed's freshness.
func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	p := params.EntityPool
	if len(p.Tokens) != 2 || len(p.Reserves) != 2 {
		return nil, ladder.ErrInvalidToken
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	proxy := strings.ToLower(staticExtra.Address)
	s := &PoolSimulator{
		Pool:        pool.FromEntity(p),
		proxy:       proxy,
		base:        strings.ToLower(p.Tokens[0].Address),
		market:      strings.ToLower(p.Tokens[1].Address),
		feed:        extra.Feed,
		feedKey:     feedKey(extra.Feed, proxy),
		feedVersion: p.Timestamp,
	}
	s.warmFeed()
	return s, nil
}

// AfterMsgpackUnmarshal warms the feed connection for simulators that
// routers decode from cache, which never pass through NewPoolSimulator.
func (s *PoolSimulator) AfterMsgpackUnmarshal() error {
	s.warmFeed()
	return nil
}

// warmFeed starts the shared connection now rather than on the first quote.
func (s *PoolSimulator) warmFeed() {
	if s.feedKey != "" {
		acquireFeed(s.feedKey, s.feed, s.proxy, s.feedVersion)
	}
}

// current resolves the state to quote on and its decay: the pinned state if
// any, else the latest quotable snapshot covering this market. It never
// blocks, dials or does RPC.
func (s *PoolSimulator) current(now time.Time) (*ladder.PoolSimulator, SwapInfo, error) {
	if s.pinned != nil {
		return s.pinned, SwapInfo{Seq: s.pinnedSeq, Decay: s.pinnedDecay}, nil
	}
	if s.feedKey == "" {
		return nil, SwapInfo{}, ErrNoFeed
	}
	snap, decay := acquireFeed(s.feedKey, s.feed, s.proxy, s.feedVersion).quotable(now)
	if state := snap.market(s.base, s.market); state != nil {
		return state, SwapInfo{Seq: snap.seq, Decay: decay}, nil
	}
	return nil, SwapInfo{}, ErrNoFeed
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	state, info, err := s.current(time.Now())
	if err != nil {
		return nil, err
	}
	result, err := state.CalcAmountOut(params)
	if err != nil {
		return nil, err
	}

	info.state, info.ladderOut = state, result.TokenAmountOut.Amount
	if info.Decay != 0 {
		// out / (1 + DecayBps/10000 * secondsStale)
		amountOut := new(big.Int).SetUint64(decayScale)
		amountOut.Mul(amountOut, info.ladderOut)
		amountOut.Quo(amountOut, new(big.Int).Add(big.NewInt(decayScale), new(big.Int).SetUint64(info.Decay)))
		if amountOut.Sign() <= 0 {
			return nil, ladder.ErrNoQuote
		}
		result.TokenAmountOut = &pool.TokenAmount{Token: result.TokenAmountOut.Token, Amount: amountOut}
	}
	result.SwapInfo = info
	return result, nil
}

// UpdateBalance applies the swap to a private copy of the exact state it
// was quoted on, even if a newer snapshot has arrived since, and pins this
// simulator there. Snapshot state is shared, so it is never mutated. The
// ladder consumes the undecayed output it priced; the shared inventory
// limit consumes what is actually paid out.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok || info.state == nil {
		state, current, err := s.current(time.Now())
		if err != nil {
			return
		}
		info, info.state = current, state
	}

	ladderParams := params
	ladderParams.SwapLimit = nil
	if info.ladderOut != nil {
		ladderParams.TokenAmountOut = pool.TokenAmount{Token: params.TokenAmountOut.Token, Amount: info.ladderOut}
	}
	next := info.state.CloneState().(*ladder.PoolSimulator)
	next.UpdateBalance(ladderParams)
	s.pinned, s.pinnedSeq, s.pinnedDecay = next, info.Seq, info.Decay

	if limit := params.SwapLimit; limit != nil {
		_, _, _ = limit.UpdateLimit(
			params.TokenAmountOut.Token,
			params.TokenAmountIn.Token,
			params.TokenAmountOut.Amount,
			params.TokenAmountIn.Amount,
		)
	}
}

// CloneState pins the clone to the snapshot and decay current at this
// moment, so a route never mixes snapshots while the base simulator moves on.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	if state, info, err := s.current(time.Now()); err == nil {
		cloned.pinned = state.CloneState().(*ladder.PoolSimulator)
		cloned.pinnedSeq, cloned.pinnedDecay = info.Seq, info.Decay
	}
	return &cloned
}

func (s *PoolSimulator) GetReserves() []*big.Int {
	if state, _, err := s.current(time.Now()); err == nil {
		return state.Info.Reserves
	}
	return s.Info.Reserves
}

// CalculateLimit reports the snapshot's deliverable inventory. One vault
// backs every market, so the shared swap limit carries consumption across
// pools. Without a quotable snapshot nothing is deliverable.
func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	if state, _, err := s.current(time.Now()); err == nil {
		return state.CalculateLimit()
	}
	return map[string]*big.Int{s.Info.Tokens[0]: new(big.Int), s.Info.Tokens[1]: new(big.Int)}
}

// GetMetaInfo keeps ApprovalAddress (rather than ladder.PoolMeta's default
// shape) so callers that read pool.ApprovalInfo off it resolve the proxy.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	blockNumber := s.Info.BlockNumber
	if state, _, err := s.current(time.Now()); err == nil && state.Info.BlockNumber != 0 {
		blockNumber = state.Info.BlockNumber
	}
	return pool.MetaInfo{ApprovalAddress: s.proxy, BlockNumber: blockNumber}
}
