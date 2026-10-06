package flywheelfun

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Curve      CurveState
	Static     StaticExtra
	Valid      bool
	MarketPool *v3.PoolSimulator
	Protocol   [2]uint32
	Base       pool.IPoolSimulator // route pool (indexed uniswap pool), nil for WETH; deep-copied on clone
	Route      []byte              // settlement route through Base
	Revision   uint64
}

var _ pool.IMetaPoolSimulator = (*PoolSimulator)(nil)
var _ = pool.RegisterFactoryMeta(DexType, NewPoolSimulatorWithBases)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	return NewPoolSimulatorWithBases(p, nil)
}
func NewPoolSimulatorWithBases(p entity.Pool, baseMap map[string]pool.IPoolSimulator) (*PoolSimulator, error) {
	var e Extra
	var st StaticExtra
	if err := json.Unmarshal([]byte(p.Extra), &e); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(p.StaticExtra), &st); err != nil {
		return nil, err
	}
	if len(p.Tokens) != 2 || p.Tokens[0] == nil || p.Tokens[1] == nil || !common.IsHexAddress(p.Address) || p.Tokens[0].Address != WETH || strings.ToLower(p.Address) != p.Tokens[1].Address || p.BlockNumber == 0 || st.Factory != Factory || st.Settlement != Settlement || !common.IsHexAddress(st.Quote) || st.Quote == p.Address {
		return nil, ErrState
	}
	if st.Quote != WETH && len(st.BasePools) == 0 || e.Curve.Graduated && e.MarketPool == nil {
		return nil, ErrUnsupported
	}
	if !e.Valid || len(st.BasePools) > 1 || st.Quote == WETH && len(st.BasePools) > 0 {
		return nil, ErrState
	}
	s := &PoolSimulator{Pool: pool.Pool{Info: pool.PoolInfo{Address: p.Address, Exchange: p.Exchange, Type: p.Type, Tokens: []string{WETH, p.Address}, BlockNumber: p.BlockNumber}}, Curve: e.Curve, Static: st, Valid: true, Protocol: e.Protocol}
	if len(st.BasePools) == 1 {
		if s.Base = baseMap[st.BasePools[0]]; s.Base == nil {
			return nil, ErrBasePool
		}
		if s.Base.GetTokenIndex(WETH) < 0 || s.Base.GetTokenIndex(st.Quote) < 0 {
			return nil, ErrState
		}
		hop, err := hopFor(s.Base)
		if err != nil {
			return nil, err
		}
		if s.Route, err = encodeRoute(hop); err != nil {
			return nil, err
		}
	}
	if e.Curve.Graduated {
		var err error
		s.MarketPool, err = v3.NewPoolSimulator(*e.MarketPool, 4663)
		if err != nil {
			return nil, err
		}
		if s.MarketPool.Info.BlockNumber != p.BlockNumber || s.MarketPool.GetTokenIndex(st.Quote) < 0 || s.MarketPool.GetTokenIndex(p.Address) < 0 || s.Protocol[0] > 1000 || s.Protocol[1] > 1000 {
			return nil, ErrState
		}
	}
	if err := s.syncReserves(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *PoolSimulator) CalcAmountOut(p pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	i, o := s.GetTokenIndex(p.TokenAmountIn.Token), s.GetTokenIndex(p.TokenOut)
	if !s.Valid || i < 0 || o < 0 || i == o || p.TokenAmountIn.Amount == nil || p.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrAmount
	}
	input, overflow := uint256.FromBig(p.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrMath
	}
	buy := i == 0
	protocol := uint32(0)
	if s.MarketPool != nil {
		in := s.Static.Quote
		if !buy {
			in = s.Info.Address
		}
		protocol = s.Protocol[s.MarketPool.GetTokenIndex(in)]
	} else if s.Curve.Graduated {
		return nil, ErrState
	}
	next := s.Curve
	gas := int64(900_000)
	steps := make([]SwapStep, 0, 3)
	var out, fee, minQuote, refund, refundRouteOutput uint256.Int
	runMarket := func(a *uint256.Int) (uint256.Int, uint256.Int, error) {
		if s.MarketPool != nil {
			in, out := s.Static.Quote, s.Info.Address
			if !buy {
				in, out = out, in
			}
			v, step, g, err := swapCL(s.MarketPool, a, in, out, protocol)
			if err == nil {
				steps = append(steps, step)
				gas += g
			}
			return v, *a, err
		}
		v, n, used, err := CurveFill(s.Curve, a, buy)
		next = n
		return v, used, err
	}
	// runRoute swaps WETH <-> quote through base; WETH-paired launches pass amounts through.
	runRoute := func(base pool.IPoolSimulator, a *uint256.Int, toQuote bool) (uint256.Int, error) {
		if base == nil {
			return *a, nil
		}
		in, out := WETH, s.Static.Quote
		if !toQuote {
			in, out = out, in
		}
		v, step, g, err := hopSwap(base, a, in, out, 0)
		if err == nil {
			steps = append(steps, step)
			gas += g
		}
		return v, err
	}
	if buy {
		platform, err := BuyPlatform(input, protocol)
		if err != nil {
			return nil, err
		}
		originalPlatform := platform
		var routingInput uint256.Int
		routingInput.Sub(input, &platform)
		quote, err := runRoute(s.Base, &routingInput, true)
		if err != nil {
			return nil, err
		}
		minQuote = quote
		split, err := BuyQuote(&quote, protocol)
		if err != nil {
			return nil, err
		}
		var used uint256.Int
		out, used, err = runMarket(&split.Net)
		if err != nil {
			return nil, err
		}
		usedGross := quote
		if !used.Eq(&split.Net) {
			if protocol != 0 {
				return nil, ErrPartial
			}
			usedGross, err = ratio(&used, 9905, 9600, true)
			if err != nil || usedGross.Gt(&quote) {
				return nil, ErrMath
			}
			split, err = BuyQuote(&usedGross, 0)
			if err != nil || !split.Net.Eq(&used) {
				return nil, ErrMath
			}
			platform, err = mulDiv(&platform, &usedGross, &quote, true)
			if err != nil {
				return nil, err
			}
			var remaining uint256.Int
			remaining.Sub(&quote, &usedGross)
			if !remaining.IsZero() {
				// The refund swaps back through the base after the forward swap moved it.
				base := s.Base
				if base != nil {
					base = base.CloneState()
					base.UpdateBalance(steps[0].Params)
				}
				refundRouteOutput, err = runRoute(base, &remaining, false)
				if err != nil {
					return nil, err
				}
				refund.Sub(&originalPlatform, &platform)
				if _, bad := refund.AddOverflow(&refund, &refundRouteOutput); bad || refund.Gt(input) {
					return nil, ErrMath
				}
			}
		}
		// ETH equivalent at this execution's actual quote conversion, excluding external DEX fees.
		var quoteFees uint256.Int
		quoteFees.Sub(&usedGross, &split.Net)
		equivalent, err := mulDiv(&routingInput, &quoteFees, &quote, false)
		if err != nil {
			return nil, err
		}
		fee.Add(&platform, &equivalent)
	} else {
		quote, _, err := runMarket(input)
		if err != nil {
			return nil, err
		}
		minQuote = quote
		reserves, err := SellQuote(&quote, protocol)
		if err != nil {
			return nil, err
		}
		realized, err := runRoute(s.Base, &reserves.Net, false)
		if err != nil {
			return nil, err
		}
		split, err := SellETH(&realized, &quote, protocol)
		if err != nil {
			return nil, err
		}
		out = split.Net
		value, err := mulDiv(&realized, &quote, &reserves.Net, false)
		if err != nil {
			return nil, err
		}
		if _, bad := fee.SubOverflow(&value, &out); bad {
			return nil, ErrMath
		}
	}
	if out.IsZero() {
		return nil, ErrAmount
	}
	if !s.Curve.Graduated && next.Graduated {
		gas += 1_800_000
	}
	info := SwapInfo{Token: s.Info.Address, Buy: buy, Route: s.Route, MinQuote: minQuote, AmountOut: out, Refund: refund, RefundRouteOutput: refundRouteOutput, Next: next, Previous: s.Curve, Revision: s.Revision, Steps: steps}
	return &pool.CalcAmountOutResult{TokenAmountOut: &pool.TokenAmount{Token: p.TokenOut, Amount: out.ToBig()}, RemainingTokenAmountIn: &pool.TokenAmount{Token: p.TokenAmountIn.Token, Amount: refund.ToBig()}, Fee: &pool.TokenAmount{Token: WETH, Amount: fee.ToBig()}, Gas: gas, SwapInfo: info}, nil
}
func (s *PoolSimulator) UpdateBalance(p pool.UpdateBalanceParams) {
	info, ok := p.SwapInfo.(SwapInfo)
	if !ok || info.Token != s.Info.Address || info.Previous != s.Curve || info.Revision != s.Revision {
		s.Valid = false
		return
	}
	// Same revision: the steps were quoted on exactly this state.
	for _, step := range info.Steps {
		if step.Index == -1 {
			s.MarketPool.UpdateBalance(step.Params)
		} else {
			s.Base.UpdateBalance(step.Params)
		}
	}
	s.Curve = info.Next
	s.Revision++
	if s.syncReserves() != nil {
		s.Valid = false
	}
	// Migration creates a new pool: refresh its actual state before quoting again.
	if s.Curve.Graduated && s.MarketPool == nil {
		s.Valid = false
	}
}

// The public pair is WETH/token. Expose real backing (no virtual reserves) in
// that order, valuing custom quote backing at route spot prices. This is only
// a reserve-value proxy, never an assertion of spendable native inventory;
// CalcAmountOut still executes every route and market leg with its price impact.
func (s *PoolSimulator) syncReserves() error {
	var quote, tokens uint256.Int
	if s.MarketPool != nil {
		ix := s.MarketPool.GetTokenIndex(s.Static.Quote)
		if quote.SetFromBig(s.MarketPool.Info.Reserves[ix]) || tokens.SetFromBig(s.MarketPool.Info.Reserves[1-ix]) {
			return ErrState
		}
	} else if s.Curve.Graduated {
		s.Info.Reserves = []*big.Int{new(big.Int), new(big.Int)}
		return nil
	} else {
		if _, underflow := quote.SubOverflow(&s.Curve.QuoteReserve, &s.Curve.VirtualQuote); underflow {
			return ErrState
		}
		tokens.Set(&s.Curve.TokenReserve)
	}
	if s.Base != nil {
		c, err := core(s.Base)
		if err != nil {
			return err
		}
		if err = quoteToWETH(&quote, &c.V3Pool.SqrtRatioX96, c.Info.Tokens[0] == s.Static.Quote); err != nil {
			return err
		}
	}
	s.Info.Reserves = []*big.Int{quote.ToBig(), tokens.ToBig()}
	return nil
}

// CloneState copies what UpdateBalance mutates in place: the base and market pools.
// Reserves are reassigned wholesale by syncReserves.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	c := *s
	if s.Base != nil {
		c.Base = s.Base.CloneState()
	}
	if s.MarketPool != nil {
		c.MarketPool = s.MarketPool.CloneState().(*v3.PoolSimulator)
	}
	return &c
}
func (s *PoolSimulator) GetBasePools() []pool.IPoolSimulator {
	if s.Base == nil {
		return nil
	}
	return []pool.IPoolSimulator{s.Base}
}
func (s *PoolSimulator) SetBasePool(p pool.IPoolSimulator) {
	if p != nil && s.Base != nil && p.GetAddress() == s.Base.GetAddress() {
		s.Base = p
		s.Revision++ // quotes taken on the old base no longer apply
	}
}
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{Settlement: Settlement, Token: s.Info.Address, BlockNumber: s.Info.BlockNumber}
}
func (s *PoolSimulator) SwapReceiveNativeIn(in, out string, chain valueobject.ChainID) bool {
	return chain == 4663 && in == WETH && out == s.Info.Address
}
func (s *PoolSimulator) SwapReturnNativeOut(in, out string, chain valueobject.ChainID) bool {
	return chain == 4663 && out == WETH && in == s.Info.Address
}
