package uscoreprop

import (
	"context"
	"math/big"
	"slices"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct{ client *ethrpc.Client }

type stateOverrides = map[common.Address]gethclient.OverrideAccount

func NewPoolTracker(_ *Config, client *ethrpc.Client) *PoolTracker {
	return &PoolTracker{client: client}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.track(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context, p entity.Pool, params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.track(ctx, p, params.Overrides)
}

// track samples both directions' quote curves and the reserves at one block, in one eth_call.
func (t *PoolTracker) track(ctx context.Context, p entity.Pool, overrides stateOverrides) (entity.Pool, error) {
	if len(p.Tokens) != 2 || p.Tokens[0] == nil || p.Tokens[1] == nil {
		return p, ErrInvalidState
	}
	var tokens [2]common.Address
	var units [2]*uint256.Int
	for i, token := range p.Tokens {
		if token.Decimals > 38 || !common.IsHexAddress(token.Address) {
			return p, ErrInvalidState
		}
		tokens[i], units[i] = common.HexToAddress(token.Address), big256.TenPow(token.Decimals)
	}
	s, err := sample(ctx, t.client, common.HexToAddress(p.Address), tokens, units, overrides)
	if err != nil {
		return p, err
	}
	if !s.Timestamp.IsInt64() || s.Timestamp.Sign() <= 0 || !s.BlockNumber.IsUint64() || s.BlockNumber.Sign() <= 0 {
		return p, ErrInvalidState
	}
	var ladders [2][]Point
	for i, q := range [2]struct {
		points, outs []*big.Int
		status       uint8
	}{{s.Points0, s.Outs0, s.Status0}, {s.Points1, s.Outs1, s.Status1}} {
		points := big256.MustFromBigs(q.points)
		quote, err := toLadderQuote(points, q.outs, q.status)
		if err != nil {
			return p, err
		}
		ladders[i] = collectQuotes(points, quote)
	}
	extra, err := json.Marshal(Extra{Ladders: ladders})
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{s.ReserveQuote.String(), s.ReserveBase.String()}
	for dir := range ladders {
		if len(ladders[dir]) == 0 { // nothing obtainable: don't report it as liquidity
			p.Reserves[1-dir] = "0"
		}
	}
	p.BlockNumber = s.BlockNumber.Uint64()
	p.Timestamp = s.Timestamp.Int64()
	return p, nil
}

// toLadderQuote validates a quoteLadder result against the sampled inputs.
func toLadderQuote(points []*uint256.Int, outs []*big.Int, status uint8) (ladderQuote, error) {
	if len(points) == 0 {
		return ladderQuote{}, nil
	}
	if status > maxStatus || len(outs) != len(points) || slices.Contains(outs, nil) {
		return ladderQuote{}, ErrInvalidState
	}
	return ladderQuote{outs: big256.MustFromBigs(outs), status: status}, nil
}

// collectQuotes keeps the quotable prefix: it stops at the first zero, decreasing or oversized output.
func collectQuotes(points []*uint256.Int, q ladderQuote) []Point {
	if q.status != statusOK && q.status != statusInsufficient {
		return nil
	}
	curve := make([]Point, 0, len(q.outs))
	for i, out := range q.outs {
		if out.IsZero() || out.Gt(maxPoint) || i > 0 && out.Lt(q.outs[i-1]) {
			break
		}
		curve = append(curve, Point{*points[i], *out})
	}
	return curve
}
