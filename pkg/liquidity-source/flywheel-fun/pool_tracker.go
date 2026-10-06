package flywheelfun

import (
	"context"
	"math/big"
	"slices"

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
	rpc *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

// NewPoolTracker tracks curve and market state only. Route hops are the indexed
// uniswapv3/uniswap-v4 pools, tracked by their own sources.
func NewPoolTracker(_ *Config, r *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{rpc: r}, nil
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
	var v3slot v3Slot
	var v4slot []common.Hash
	if len(s.BasePools) == 1 {
		if id := s.BasePools[0]; common.IsHexAddress(id) {
			req.AddCall(&ethrpc.Call{ABI: stateABI, Target: id, Method: "slot0"}, []any{&v3slot})
		} else {
			root := mappingSlot(common.HexToHash(id).Big(), common.BigToHash(big.NewInt(6)))
			req.AddCall(&ethrpc.Call{ABI: stateABI, Target: Manager, Method: "extsload", Params: []any{[]common.Hash{root}}}, []any{&v4slot})
		}
	}
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
	e := Extra{Curve: state, Valid: s.Quote == WETH || len(s.BasePools) > 0, Dependencies: []string{Factory, MarketAdapter, Manager}}
	if c.Graduated && e.Valid {
		market, protocol, er := t.canonical(ctx, p.Address, c, head.Number)
		if er != nil {
			return p, er
		}
		e.MarketPool = &market
		e.Protocol = protocol
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
		var quote, tokens uint256.Int
		if _, overflow := quote.SubOverflow(&state.QuoteReserve, &state.VirtualQuote); overflow {
			return p, ErrState
		}
		tokens.Set(&state.TokenReserve)
		if e.MarketPool != nil {
			ix := 0
			if e.MarketPool.Tokens[0].Address != s.Quote {
				ix = 1
			}
			if quote.SetFromDecimal(e.MarketPool.Reserves[ix]) != nil || tokens.SetFromDecimal(e.MarketPool.Reserves[1-ix]) != nil {
				return p, ErrState
			}
		}
		if len(s.BasePools) == 1 {
			var price uint256.Int
			// ponytail: assumes a V4 base's WETH side is native ETH (currency0); an ERC20-WETH V4
			// pool with quote < WETH inverts this reserve proxy. Router quotes use the real key.
			quoteIsToken0 := s.Quote < WETH && common.IsHexAddress(s.BasePools[0])
			if len(v4slot) == 1 {
				price.SetBytes(v4slot[0][12:]) // sqrtPriceX96 is slot0's low 160 bits
			} else if v3slot.SqrtPriceX96 == nil || price.SetFromBig(v3slot.SqrtPriceX96) {
				return p, ErrState
			}
			if err = quoteToWETH(&quote, &price, quoteIsToken0); err != nil {
				return p, err
			}
		}
		p.Reserves = entity.PoolReserves{quote.Dec(), tokens.Dec()}
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
