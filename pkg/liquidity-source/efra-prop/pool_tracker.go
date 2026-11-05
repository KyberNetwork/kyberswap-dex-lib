package efraprop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPoolState probes quoteExactIn across both directions. The venue's curve (bins,
// cursors, gap, flat USDC fee, gap discount) is only published on-chain, and its quotes
// expire 15 blocks after the maker's last update, so the ladder is re-probed every cycle
// instead of being derived from tokenConfigs/prices/getBins.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidToken
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	venue := common.HexToAddress(staticExtra.Venue)
	tokens := [2]common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}

	// Balances are the only reserve the venue exposes (it holds every market's inventory),
	// and they size the probe grid and cap each direction's payout.
	var balances [2]*big.Int
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	for i, token := range tokens {
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: token.Hex(), Method: "balanceOf",
			Params: []any{venue}}, []any{&balances[i]})
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	points := [2][]*big.Int{
		ladder.SamplePoints(p, 0, balances[0], balances[1]),
		ladder.SamplePoints(p, 1, balances[1], balances[0]),
	}
	outputs, err := t.probeQuotes(ctx, venue, tokens, points, common.HexToAddress(staticExtra.Taker), resp.BlockNumber)
	if err != nil {
		return p, err
	}

	for dir := range outputs {
		t.applyBuffer(outputs[dir])
	}
	ladders := [2][]ladder.Point{
		ladder.CollectLadder(points[0], outputs[0]),
		ladder.CollectLadder(points[1], outputs[1]),
	}

	extraBytes, err := json.Marshal(ladder.Extra{Ladders: ladders})
	if err != nil {
		return p, err
	}

	// quoteExactIn answers 0 for paused, stale (block.number >= quoteStaleBlock), disabled or
	// out-of-band markets, so an empty side is exactly the venue's "not tradeable".
	p.Extra = string(extraBytes)
	p.Reserves = ladder.ZeroUnquotedReserves(
		entity.PoolReserves{balStr(balances[0]), balStr(balances[1])}, ladders)
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// probeQuotes pins every probe to one block: the venue's answer moves with each maker
// update, and mixing blocks would build a ladder out of two different curves.
func (t *PoolTracker) probeQuotes(
	ctx context.Context, venue common.Address, tokens [2]common.Address,
	points [2][]*big.Int, taker common.Address, blockNumber *big.Int,
) ([2][]*big.Int, error) {
	var outputs [2][]*big.Int
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	callCount := 0
	for dir := range points {
		tokenIn, tokenOut := tokens[dir], tokens[1-dir]
		outputs[dir] = make([]*big.Int, len(points[dir]))
		for i, pt := range points[dir] {
			outputs[dir][i] = new(big.Int)
			req.AddCall(&ethrpc.Call{ABI: venueABI, Target: venue.Hex(), Method: "quoteExactIn",
				Params: []any{tokenIn, tokenOut, pt, taker}}, []any{&outputs[dir][i]})
			callCount++
		}
	}
	if callCount == 0 {
		return outputs, nil
	}
	if _, err := req.TryAggregate(); err != nil {
		return outputs, err
	}
	return outputs, nil
}

func (t *PoolTracker) applyBuffer(results []*big.Int) {
	if t.cfg.Buffer <= 0 {
		return
	}
	buf := big.NewInt(t.cfg.Buffer)
	for _, r := range results {
		if r != nil {
			bignumber.MulDivDown(r, r, buf, bignumber.BasisPoint)
		}
	}
}

func balStr(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}
