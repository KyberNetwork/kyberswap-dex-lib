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
	Curve         CurveState
	Static        StaticExtra
	Valid         bool
	MarketPool    *v3.PoolSimulator
	Protocol      [2]uint32
	Route         []RouteHop
	RouteEntities []entity.Pool
	Bases         []pool.IPoolSimulator
	Revision      uint64
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
	if st.Quote != WETH && len(e.Route) == 0 || e.Curve.Graduated && e.MarketPool == nil {
		return nil, ErrUnsupported
	}
	if !e.Valid {
		return nil, ErrState
	}
	if len(e.Route) > 3 || len(e.Route) != len(e.RoutePools) || st.Quote == WETH && len(e.Route) > 0 {
		return nil, ErrState
	}
	s := &PoolSimulator{Pool: pool.Pool{Info: pool.PoolInfo{Address: p.Address, Exchange: p.Exchange, Type: p.Type, Tokens: []string{WETH, p.Address}, Reserves: []*big.Int{e.Curve.QuoteReserve.ToBig(), e.Curve.TokenReserve.ToBig()}, BlockNumber: p.BlockNumber}}, Curve: e.Curve, Static: st, Valid: true, Protocol: e.Protocol, Route: e.Route, RouteEntities: e.RoutePools}
	current := WETH
	visited := map[string]bool{WETH: true}
	for i, h := range e.Route {
		id, err := h.id()
		if err != nil {
			return nil, err
		}
		base := baseMap[id]
		if base == nil {
			base, err = routePool(e.RoutePools[i], h)
			if err != nil {
				return nil, err
			}
		}
		if err = validateBase(base, e.RoutePools[i], h, p.BlockNumber); err != nil {
			return nil, err
		}
		c, _ := core(base)
		ix := c.GetTokenIndex(current)
		if ix < 0 {
			return nil, ErrState
		}
		current = c.Info.Tokens[1-ix]
		if visited[current] || current == p.Address {
			return nil, ErrState
		}
		visited[current] = true
		s.Bases = append(s.Bases, base)
	}
	if current != st.Quote {
		return nil, ErrState
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
	var market *v3.PoolSimulator
	if s.Curve.Graduated {
		if s.MarketPool == nil {
			return nil, ErrState
		}
		market = s.MarketPool.CloneState().(*v3.PoolSimulator)
		in := s.Static.Quote
		if !buy {
			in = s.Info.Address
		}
		protocol = s.Protocol[market.GetTokenIndex(in)]
	}
	bases := cloneBases(s.Bases)
	next := s.Curve
	gas := int64(900_000)
	var steps []SwapStep
	var out, fee, minQuote, refund, refundRouteOutput uint256.Int
	runMarket := func(a *uint256.Int) (uint256.Int, uint256.Int, error) {
		if market != nil {
			in, out := s.Static.Quote, s.Info.Address
			if !buy {
				in, out = out, in
			}
			v, step, g, err := swapCL(market, a, in, out, protocol, -1)
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
	runRoute := func(a *uint256.Int, direction bool) (uint256.Int, error) {
		v, st, g, err := s.routeSwap(a, direction, bases)
		steps = append(steps, st...)
		gas += g
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
		quote, err := runRoute(&routingInput, true)
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
				refundRouteOutput, err = runRoute(&remaining, false)
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
		realized, err := runRoute(&reserves.Net, false)
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
	info := SwapInfo{Next: next, Previous: s.Curve, Token: s.Info.Address, MinQuote: minQuote.Clone(), AmountOut: out.Clone(), Refund: refund.Clone(), RefundRouteOutput: refundRouteOutput.Clone(), Buy: buy, Route: copyRoutes(s.Route), Steps: steps, Revision: s.Revision}
	return &pool.CalcAmountOutResult{TokenAmountOut: &pool.TokenAmount{Token: p.TokenOut, Amount: out.ToBig()}, RemainingTokenAmountIn: &pool.TokenAmount{Token: p.TokenAmountIn.Token, Amount: refund.ToBig()}, Fee: &pool.TokenAmount{Token: WETH, Amount: fee.ToBig()}, Gas: gas, SwapInfo: info}, nil
}
func (s *PoolSimulator) UpdateBalance(p pool.UpdateBalanceParams) {
	info, ok := p.SwapInfo.(SwapInfo)
	if !ok || info.Token != s.Info.Address || info.Previous != s.Curve || info.Revision != s.Revision {
		s.Valid = false
		return
	}
	// Validate the entire replay on clones before mutating any shared liquidity.
	bases := cloneBases(s.Bases)
	var market pool.IPoolSimulator
	if s.MarketPool != nil {
		market = s.MarketPool.CloneState()
	}
	target := func(index int, base []pool.IPoolSimulator, m pool.IPoolSimulator) pool.IPoolSimulator {
		if index == -1 {
			return m
		}
		if index < 0 || index >= len(base) {
			return nil
		}
		return base[index]
	}
	for _, step := range info.Steps {
		dest := target(step.Index, bases, market)
		if dest == nil || fingerprint(dest) != step.Before {
			s.Valid = false
			return
		}
		dest.UpdateBalance(step.Params)
	}
	for _, step := range info.Steps {
		target(step.Index, s.Bases, s.MarketPool).UpdateBalance(step.Params)
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
	var quote, tokens *big.Int
	if s.MarketPool != nil {
		ix := s.MarketPool.GetTokenIndex(s.Static.Quote)
		quote = new(big.Int).Set(s.MarketPool.Info.Reserves[ix])
		tokens = new(big.Int).Set(s.MarketPool.Info.Reserves[1-ix])
	} else if s.Curve.Graduated {
		s.Info.Reserves = []*big.Int{new(big.Int), new(big.Int)}
		return nil
	} else {
		quote = new(big.Int).Sub(s.Curve.QuoteReserve.ToBig(), s.Curve.VirtualQuote.ToBig())
		tokens = s.Curve.TokenReserve.ToBig()
		if quote.Sign() < 0 {
			return ErrState
		}
	}
	current := s.Static.Quote
	q192 := new(big.Int).Lsh(big.NewInt(1), 192)
	for i := len(s.Bases) - 1; i >= 0; i-- {
		c, err := core(s.Bases[i])
		if err != nil {
			return err
		}
		ix := c.GetTokenIndex(current)
		if ix < 0 {
			return ErrState
		}
		price := c.V3Pool.SqrtRatioX96.ToBig()
		price.Mul(price, price)
		if price.Sign() == 0 {
			return ErrState
		}
		if ix == 0 {
			quote.Mul(quote, price)
			quote.Div(quote, q192)
		} else {
			quote.Mul(quote, q192)
			quote.Div(quote, price)
		}
		current = c.Info.Tokens[1-ix]
	}
	if current != WETH || quote.BitLen() > 256 {
		return ErrState
	}
	s.Info.Reserves = []*big.Int{quote, tokens}
	return nil
}
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	c := *s
	c.Info.Tokens = append([]string(nil), s.Info.Tokens...)
	c.Info.Reserves = cloneReserves(s.Info.Reserves)
	c.Route = copyRoutes(s.Route)
	c.Bases = cloneBases(s.Bases)
	if s.MarketPool != nil {
		c.MarketPool = s.MarketPool.CloneState().(*v3.PoolSimulator)
	}
	return &c
}
func (s *PoolSimulator) GetBasePools() []pool.IPoolSimulator {
	return append([]pool.IPoolSimulator(nil), s.Bases...)
}
func (s *PoolSimulator) SetBasePool(p pool.IPoolSimulator) {
	if p == nil {
		return
	}
	for i, b := range s.Bases {
		if b.GetAddress() == p.GetAddress() {
			if validateBase(p, s.RouteEntities[i], s.Route[i], s.Info.BlockNumber) != nil {
				s.Valid = false
				return
			}
			s.Bases[i] = p
		}
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
