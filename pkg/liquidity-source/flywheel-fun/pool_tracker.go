package flywheelfun

import (
	"context"
	"math/big"
	"slices"
	"sync"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type CurveRPC struct {
	Creator, QuoteAsset                                                                                         common.Address
	QuoteReserve, TokenReserve, VirtualQuoteOffset, GraduationThreshold, LaunchTimestamp, SnipeTaxWindowSeconds *big.Int
	SnipeTaxStartBps, CreatorFeeBps, ProtocolFeeBps, TradingFeeBps                                              uint16
	Graduated                                                                                                   bool
	Pool, Token0, Token1                                                                                        common.Address
	PositionTokenId                                                                                             *big.Int
}
type PoolTracker struct {
	rpc        *ethrpc.Client
	config     *Config
	routeMu    sync.Mutex
	routeBlock common.Hash
	routeCache map[string]routeSnapshot
}
type routeSnapshot struct {
	pool entity.Pool
	hop  RouteHop
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, r *ethrpc.Client) (*PoolTracker, error) {
	if config == nil {
		config = &Config{}
	}
	copyConfig := *config
	copyConfig.QuoteRoutes = make(map[string][]RouteHop, len(config.QuoteRoutes))
	for quote, route := range config.QuoteRoutes {
		if !common.IsHexAddress(quote) || len(route) > 3 {
			return nil, ErrState
		}
		address := common.HexToAddress(quote)
		if address == (common.Address{}) {
			return nil, ErrState
		}
		copyConfig.QuoteRoutes[hexutil.Encode(address[:])] = copyRoutes(route)
	}
	return &PoolTracker{rpc: r, config: &copyConfig}, nil
}
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	var s StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &s); err != nil {
		return p, err
	}
	if s.Factory != Factory || s.Settlement != Settlement || !common.IsHexAddress(p.Address) {
		return p, ErrState
	}
	chain, err := t.rpc.GetETHClient().ChainID(ctx)
	if err != nil {
		return p, err
	}
	if chain.Cmp(big.NewInt(4663)) != 0 {
		return p, ErrState
	}
	head, err := t.rpc.GetETHClient().HeaderByNumber(ctx, nil)
	if err != nil {
		return p, err
	}
	var c CurveRPC
	var f, a common.Address
	var held *big.Int
	req := t.rpc.NewRequest().SetContext(ctx).SetBlockNumber(head.Number)
	req.AddCall(&ethrpc.Call{ABI: factoryABI, Target: Factory, Method: "curves", Params: []any{common.HexToAddress(p.Address)}}, []any{&c})
	req.AddCall(&ethrpc.Call{ABI: factoryABI, Target: Factory, Method: "nativeMarketAdapter"}, []any{&a})
	req.AddCall(&ethrpc.Call{ABI: tokenABI, Target: p.Address, Method: "factory"}, []any{&f})
	req.AddCall(&ethrpc.Call{ABI: tokenABI, Target: p.Address, Method: "balanceOf", Params: []any{common.HexToAddress(Factory)}}, []any{&held})
	if _, err = req.Aggregate(); err != nil {
		return p, err
	}
	if hexutil.Encode(f[:]) != Factory || hexutil.Encode(a[:]) != MarketAdapter || hexutil.Encode(c.QuoteAsset[:]) != s.Quote || c.TradingFeeBps != 0 {
		return p, ErrState
	}
	state := CurveState{Graduated: c.Graduated}
	for _, pair := range []struct {
		src *big.Int
		dst *uint256.Int
	}{{c.QuoteReserve, &state.QuoteReserve}, {c.TokenReserve, &state.TokenReserve}, {c.VirtualQuoteOffset, &state.VirtualQuote}, {c.GraduationThreshold, &state.Threshold}} {
		if pair.src == nil || pair.src.Sign() < 0 || pair.dst.SetFromBig(pair.src) {
			return p, ErrState
		}
	}
	if held == nil || (!c.Graduated && held.Cmp(c.TokenReserve) < 0) {
		return p, ErrState
	}
	e := Extra{Curve: state, Valid: s.Quote == WETH, Dependencies: []string{Factory, MarketAdapter, Manager}}
	if c.Graduated && (s.Quote == WETH || len(t.config.QuoteRoutes[s.Quote]) > 0) {
		market, protocol, er := t.canonical(ctx, p.Address, c, head.Number)
		if er != nil {
			return p, er
		}
		e.MarketPool = &market
		e.Protocol = protocol
	}
	if s.Quote != WETH {
		e.Route = copyRoutes(t.config.QuoteRoutes[s.Quote])
		if len(e.Route) > 3 {
			return p, ErrState
		}
		for i, h := range e.Route {
			var base entity.Pool
			var er error
			base, h, er = t.cachedRoute(ctx, h, head.Number, head.Hash())
			if er != nil {
				return p, er
			}
			e.Route[i] = h
			e.RoutePools = append(e.RoutePools, base)
			e.Dependencies = append(e.Dependencies, base.Address)
		}
		e.Valid = len(e.Route) > 0
	}
	var previous Extra
	if json.Unmarshal([]byte(p.Extra), &previous) == nil && slices.Equal(previous.Dependencies, e.Dependencies) {
		e.DependenciesStored = previous.DependenciesStored
	}
	extra, err := json.Marshal(e)
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.BlockNumber = head.Number.Uint64()
	p.Timestamp = int64(head.Time)
	p.Reserves = entity.PoolReserves{"0", "0"}
	if e.Valid {
		var real uint256.Int
		if _, overflow := real.SubOverflow(&state.QuoteReserve, &state.VirtualQuote); overflow {
			return p, ErrState
		}
		p.Reserves = entity.PoolReserves{real.Dec(), state.TokenReserve.Dec()}
		if e.MarketPool != nil {
			if e.MarketPool.Tokens[0].Address == s.Quote {
				p.Reserves = append(entity.PoolReserves(nil), e.MarketPool.Reserves...)
			} else {
				p.Reserves = entity.PoolReserves{e.MarketPool.Reserves[1], e.MarketPool.Reserves[0]}
			}
		}
		var sim *PoolSimulator
		if sim, err = NewPoolSimulator(p); err != nil {
			return p, err
		}
		p.Reserves = entity.PoolReserves{sim.Info.Reserves[0].String(), sim.Info.Reserves[1].String()}
	}
	return p, nil
}

func (t *PoolTracker) GetDependencies(_ context.Context, p entity.Pool) ([]string, bool, error) {
	var e Extra
	if err := json.Unmarshal([]byte(p.Extra), &e); err != nil {
		return nil, false, err
	}
	return append([]string(nil), e.Dependencies...), e.DependenciesStored, nil
}
func (t *PoolTracker) SetDependenciesStored(p *entity.Pool, stored bool) error {
	var e Extra
	if err := json.Unmarshal([]byte(p.Extra), &e); err != nil {
		return err
	}
	e.DependenciesStored = stored
	b, err := json.Marshal(e)
	if err == nil {
		p.Extra = string(b)
	}
	return err
}
