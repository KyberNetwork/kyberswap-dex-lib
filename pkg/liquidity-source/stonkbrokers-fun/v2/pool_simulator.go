package stonkbrokersfunv2

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator prices ONE (pad, launchId) Smart Launch V2 launch. Buy only
// (quote -> project token) -- see constant.go's scope-decision comment and
// ErrSellNotSupported for why sell is not implemented.
type PoolSimulator struct {
	pool.Pool

	staticExtra StaticExtra
	extra       Extra

	loadedSupply *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidPoolTokens
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	loadedSupply, err := uint256.FromDecimal(staticExtra.LoadedSupply)
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bignum.NewBig(s) }),
			BlockNumber: ep.BlockNumber,
		}},
		staticExtra:  staticExtra,
		extra:        extra,
		loadedSupply: loadedSupply,
	}, nil
}

// CalcAmountOut supports exactly one direction: token1 (quote) -> token0
// (project token). Every gate StonkSafeLaunchpadV2.buy()/_buy() would revert
// on is checked here first, per AGENTS.md ("track every flag/check on the
// swap path so the simulator can reject swaps that would revert on-chain").
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenOut)
	if indexIn != 1 || indexOut != 0 {
		// Either an unknown token, or the token0->token1 (sell) direction,
		// which this integration deliberately does not implement.
		if indexIn == 0 && indexOut == 1 {
			return nil, ErrSellNotSupported
		}
		return nil, ErrInvalidToken
	}

	if s.extra.Aborted {
		return nil, ErrPoolAborted
	}
	if s.extra.Bonded {
		return nil, ErrPoolBonded
	}
	if s.extra.Graduated {
		return nil, ErrPoolGraduated
	}
	if !s.extra.Armed {
		return nil, ErrPoolNotArmed
	}
	now := uint64(time.Now().Unix())
	if !s.staticExtra.OpenEnded && now >= s.staticExtra.Deadline {
		return nil, ErrWindowClosed
	}
	// _tradeGates' last check. An aggregator route reaches buy() from the
	// executor contract, so msg.sender != tx.origin always holds and an
	// eoaOnly launch reverts NotEoa() unconditionally -- refuse rather than
	// quote a guaranteed revert.
	if s.staticExtra.EoaOnly {
		return nil, ErrEoaOnly
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn == nil {
		return nil, ErrZeroTrade
	}

	taxBps := CurrentTaxBps(now, s.staticExtra)
	tokensOut, tax, newVQuote, newVToken, err := CalcBuyAmountOut(amountIn, s.extra.VQuote, s.extra.VToken, taxBps)
	if err != nil {
		return nil, err
	}

	// _buy credits boughtOf[id][recipient] += tokensOut and then reverts
	// BuyCapExceeded() if the running total passes the cap. We cannot see the
	// recipient's existing position, so enforce the single-trade upper bound:
	// a tokensOut already past the cap can never execute for any recipient.
	if s.staticExtra.MaxBuyPpm != 0 {
		var ppm, capTokens uint256.Int
		ppm.SetUint64(uint64(s.staticExtra.MaxBuyPpm))
		capTokens.Mul(s.loadedSupply, &ppm)
		capTokens.Div(&capTokens, ppmU256)
		if tokensOut.Gt(&capTokens) {
			return nil, ErrBuyCapExceeded
		}
	}

	// Buy-side graduation/StalePrice gate: StonkSafeLaunchpadV2._buy calls
	// mcapUsd8(id) UNCONDITIONALLY (no try/catch) on every successful buy to
	// decide whether to close the curve -- if the pad's oracle is stale, the
	// on-chain call reverts entirely, so this must reject the quote too, not
	// just the graduation decision.
	quoteUsd8, err := s.currentQuoteUsd8(now)
	if err != nil {
		return nil, err
	}
	mcap := McapUsd8FromReserves(newVQuote, newVToken, s.loadedSupply, quoteUsd8, s.staticExtra.QuoteDecimals)
	graduates := mcap.Cmp(uint256.NewInt(s.staticExtra.GradMcapUsd8)) >= 0

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: tokensOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: tax.ToBig()},
		Gas:            defaultGas,
		SwapInfo: SwapInfo{
			NewVQuote: newVQuote,
			NewVToken: newVToken,
			Graduates: graduates,
		},
	}, nil
}

// currentQuoteUsd8 resolves the pad's USD mark via whichever oracle path is
// wired (StaticExtra.QuoteUsdFeed direct feed, or StaticExtra.TwapPool),
// mutually exclusive per the pad's own constructor invariant.
func (s *PoolSimulator) currentQuoteUsd8(now uint64) (uint64, error) {
	if s.staticExtra.QuoteUsdFeed != "" {
		if s.extra.DirectFeed == nil {
			return 0, ErrBadOracleAnswer
		}
		return DirectFeedUsd8(*s.extra.DirectFeed, now)
	}
	if s.staticExtra.TwapPool != "" {
		if s.extra.Twap == nil {
			return 0, ErrBadOracleAnswer
		}
		return TwapQuoteUsd8(*s.extra.Twap, s.staticExtra.TwapWindowSecs, s.staticExtra.QuoteIsToken0, s.staticExtra.QuoteDecimals, now)
	}
	return 0, ErrBadOracleAnswer
}

// UpdateBalance consumes the SwapInfo CalcAmountOut already computed --
// never recomputes the swap (AGENTS.md).
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok || swapInfo.NewVQuote == nil || swapInfo.NewVToken == nil {
		return
	}
	s.extra.VQuote = swapInfo.NewVQuote
	s.extra.VToken = swapInfo.NewVToken
	if swapInfo.Graduates {
		s.extra.Graduated = true
	}
}

// CloneState deep-copies every slice/pointer UpdateBalance writes in place
// (AGENTS.md). VQuote/VToken are replaced wholesale by UpdateBalance
// (pointer reassignment, not in-place mutation), but clone them anyway so a
// clone and its source never alias the same *uint256.Int across a
// subsequent UpdateBalance on either one.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	if s.extra.VQuote != nil {
		cloned.extra.VQuote = new(uint256.Int).Set(s.extra.VQuote)
	}
	if s.extra.VToken != nil {
		cloned.extra.VToken = new(uint256.Int).Set(s.extra.VToken)
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		Pad:             s.staticExtra.Pad,
		LaunchID:        s.staticExtra.LaunchID,
		ApprovalAddress: s.staticExtra.Pad,
		BlockNumber:     s.Info.BlockNumber,
	}
}
