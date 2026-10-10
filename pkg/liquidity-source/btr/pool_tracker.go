package btr

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

const (
	defaultSamplePoints = 6
	maxSamplePoints     = 16
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

// PoolTracker refreshes a BTR pool every block: it caches the per-token risk
// flags and probes `getSwapQuote` over a small grid per ordered pair to build a
// ladder. Monad blockMs is ~400ms and the keeper mark heartbeat is independent
// of trades, so a lazy tracker would quote stale marks.
type PoolTracker struct {
	config *Config
	client *ethrpc.Client
}

func NewPoolTracker(cfg *Config, client *ethrpc.Client) *PoolTracker {
	if cfg == nil {
		cfg = NewConfig(ChainIDMonad)
	}
	return &PoolTracker{config: cfg, client: client}
}

// rpcSwapQuote mirrors IPool.SwapQuote field for field so ethrpc unpacks the
// tuple. Only AmountOut is consumed; the rest is retained for parity/debug.
type rpcSwapQuote struct {
	AmountOut  *big.Int         `abi:"amountOut"`
	AmountIn   *big.Int         `abi:"amountIn"`
	SpreadPbps uint16           `abi:"spreadPbps"`
	ProtoFee   *big.Int         `abi:"protoFee"`
	LpFee      *big.Int         `abi:"lpFee"`
	SkewIn     int8             `abi:"skewIn"`
	SkewOut    int8             `abi:"skewOut"`
	MarkPrice  *big.Int         `abi:"markPrice"`
	MidPrice   *big.Int         `abi:"midPrice"`
	CovToll    *big.Int         `abi:"covToll"`
	RouteHops  []common.Address `abi:"routeHops"`
	HopAmounts []*big.Int       `abi:"hopAmounts"`
	HopPrices  []*big.Int       `abi:"hopPrices"`
}

// rpcSwapQuoteResult wraps the single unnamed tuple output: go-ethereum copies
// a lone return value into the first field of the target struct, so the tuple
// must sit one level down (same pattern as baseline's rpcGetQuoteStateResult).
type rpcSwapQuoteResult struct {
	Quote rpcSwapQuote
}

type sampledCall struct {
	key    string
	amount *big.Int
	res    *rpcSwapQuoteResult
}

// GetNewPoolState reads flags and re-probes the swap curve at the latest block
// and stamps the block number so downstream can detect staleness.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) < 2 || t.client == nil {
		return p, ErrBTRBadState
	}
	ctx, cancel := context.WithTimeout(ctx, RPCTimeout)
	defer cancel()

	req := t.client.NewRequest().SetContext(ctx)

	flagOut := make([]*uint16, len(p.Tokens))
	for i, tok := range p.Tokens {
		if tok == nil || !common.IsHexAddress(tok.Address) {
			return p, ErrBTRBadState
		}
		flagOut[i] = new(uint16)
		req.AddCall(&ethrpc.Call{
			ABI: poolABI, Target: p.Address, Method: "getRiskFlags",
			Params: []any{common.HexToAddress(tok.Address)},
		}, []any{flagOut[i]})
	}

	n := t.config.SamplePoints
	if n <= 0 {
		n = defaultSamplePoints
	}
	if n > maxSamplePoints {
		n = maxSamplePoints
	}

	var calls []*sampledCall
	for i := range p.Tokens {
		for j := range p.Tokens {
			if i == j {
				continue
			}
			key := pairKey(p.Tokens[i].Address, p.Tokens[j].Address)
			for _, amount := range sampleAmounts(p.Tokens[i].Decimals, n) {
				q := new(rpcSwapQuoteResult)
				req.AddCall(&ethrpc.Call{
					ABI: poolABI, Target: p.Address, Method: "getSwapQuote",
					Params: []any{
						common.HexToAddress(p.Tokens[i].Address),
						common.HexToAddress(p.Tokens[j].Address),
						amount,
					},
				}, []any{q})
				calls = append(calls, &sampledCall{key: key, amount: amount, res: q})
			}
		}
	}

	// TryBlockAndAggregate tolerates individual reverts (a halted leg, an
	// out-of-range probe) instead of failing the whole refresh; resp.Result
	// carries one success flag per call in request order.
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	flags := make(map[string]uint16, len(p.Tokens))
	for i, tok := range p.Tokens {
		if i < len(resp.Result) && resp.Result[i] {
			flags[lowerAddr(tok.Address)] = *flagOut[i]
		}
	}

	ladders := make(map[string][]Point)
	offset := len(p.Tokens)
	curKey := ""
	var pts []Point
	flush := func() {
		if len(pts) > 0 {
			ladders[curKey] = pts
		}
		pts = nil
	}
	for k, c := range calls {
		if offset+k < len(resp.Result) && !resp.Result[offset+k] {
			// A revert breaks the monotonicity of this pair's ladder: stop it
			// here and drop the rest of the pair.
			flush()
			continue
		}
		out := c.res.Quote.AmountOut
		if out == nil || out.Sign() <= 0 {
			continue
		}
		if c.key != curKey {
			flush()
			curKey = c.key
		}
		if len(pts) > 0 {
			prev, ok := new(big.Int).SetString(pts[len(pts)-1].Out, 10)
			if ok && out.Cmp(prev) < 0 {
				// Output must be non-decreasing in amountIn; a decrease means
				// the samples straddle a state change, so stop the ladder.
				continue
			}
		}
		pts = append(pts, Point{In: c.amount.String(), Out: out.String()})
	}
	flush()

	extra := Extra{BlockNumber: p.BlockNumber, Flags: flags, Ladders: ladders}
	if resp.BlockNumber != nil {
		extra.BlockNumber = resp.BlockNumber.Uint64()
	}
	b, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(b)
	p.BlockNumber = extra.BlockNumber
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// sampleAmounts builds an ascending probe grid for a token with `decimals`,
// from a hundredth of a unit up to a thousand units. The core book is
// multi-asset and thin at the tails, so out-of-range probes return zero and are
// dropped when the ladder is collected.
func sampleAmounts(decimals uint8, n int) []*big.Int {
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	hundred := big.NewInt(100)
	multipliers := []int64{1, 5, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000}
	amounts := make([]*big.Int, 0, n)
	seen := make(map[string]struct{}, n)
	for _, m := range multipliers {
		if len(amounts) >= n {
			break
		}
		amount := new(big.Int).Mul(base, big.NewInt(m))
		amount.Div(amount, hundred)
		if amount.Sign() <= 0 {
			amount = big.NewInt(1)
		}
		s := amount.String()
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		amounts = append(amounts, amount)
	}
	return amounts
}
