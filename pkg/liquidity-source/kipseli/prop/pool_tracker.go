package prop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	titanClients []*rpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	t := &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}
	if cfg.usesTitan() {
		t.titanClients = titan.NewClients(cfg.Titan)
	}
	return t
}

// GetNewPoolState probes both directions in one KipseliPropLens call, which
// also returns the fresh balances and caps. The sample grid is therefore sized
// from the previous cycle's reserves and caps (already in p); the first cycle
// needs neither (decimals sweep). The pAMM variant overrides Titan's pushed
// state into that same call.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var titanState titan.State
	if t.cfg.usesTitan() {
		titanState = titan.FetchState(ctx, t.titanClients, t.cfg.Titan.Timeout)
	}

	tokens := []common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}
	amountsIn := samplePoints(p)

	snap, err := fetchSnapshot(ctx, t.ethrpcClient, common.HexToAddress(staticExtra.RouterAddress),
		t.cfg.dest(), tokens, amountsIn[:], titanState)
	if err != nil {
		return p, err
	}
	if snap.Quoter == (common.Address{}) {
		return p, ErrNoQuoter
	}

	var ladders [2][]ladder.Point
	for dir := range 2 {
		outs := snap.AmountsOut[dir]
		trimAtWalletCap(outs, snap.Balances[1-dir])
		t.applyBuffer(outs)
		ladders[dir] = ladder.CollectLadder(amountsIn[dir], outs)
	}

	// Without a fresh override the pAMM quoter reverts every quote; persisting
	// anyway would also drop SO, which downstream uses to tell the variants apart.
	if t.cfg.usesTitan() && (len(titanState.Overrides) == 0 || allEmpty(ladders)) {
		logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "pool": p.Address}).
			Warn("all quotes returned 0 (price stale), skipping pool update")
		return p, nil
	}

	extraBytes, err := json.Marshal(Extra{
		Ladders: ladders,
		SO:      titanState.ToStateOverrides(),
		Caps:    [2]string{balStr(snap.Caps[0]), balStr(snap.Caps[1])},
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{balStr(snap.Balances[0]), balStr(snap.Balances[1])}
	p.BlockNumber = snap.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// samplePoints builds each direction's probe grid from the previous cycle's
// reserves and caps, bounded like the venue itself: by the input token's
// balance, or its remaining cap room when that is tighter.
func samplePoints(p entity.Pool) [2][]*big.Int {
	var prevExtra Extra
	_ = json.Unmarshal([]byte(p.Extra), &prevExtra)

	var balances, caps [2]*big.Int
	for i := range 2 {
		if i < len(p.Reserves) {
			balances[i] = bignumber.NewBig10(p.Reserves[i])
		}
		caps[i] = bignumber.NewBig10(prevExtra.Caps[i])
	}

	var points [2][]*big.Int
	for dir := range 2 {
		bound := sampleBound(balances[dir], maxIn(balances[dir], caps[dir]))
		points[dir] = ladder.SamplePoints(p, dir, bound, balances[1-dir])
	}
	return points
}

// trimAtWalletCap zeroes every probe after the first whose output reached the
// wallet's tokenOut balance: SwapImpl caps quotes there, so the rest is a flat
// plateau the simulator would otherwise price as a real (terrible) rate.
func trimAtWalletCap(outs []*big.Int, walletBalanceOut *big.Int) {
	if walletBalanceOut == nil || walletBalanceOut.Sign() <= 0 {
		return
	}
	for j, out := range outs {
		if out != nil && out.Cmp(walletBalanceOut) >= 0 {
			clear(outs[j+1:])
			return
		}
	}
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

// sampleBound picks the tighter of the venue's own balance and its published
// cap room as the grid's ceiling — a token with no cap published falls back
// to its balance.
func sampleBound(balance, maxIn *big.Int) *big.Int {
	bound := balance
	if maxIn != nil && maxIn.Sign() > 0 && (bound == nil || maxIn.Cmp(bound) < 0) {
		bound = maxIn
	}
	return bound
}

// maxIn reports how much of a token the venue can still absorb before its
// published cap. No cap published (nil, zero, or max-uint) yields nil, which
// keeps the grid falling back to the raw balance.
func maxIn(balance, cap_ *big.Int) *big.Int {
	if cap_ == nil || cap_.Sign() <= 0 || cap_.Cmp(bignumber.MaxUint256) == 0 {
		return nil
	}
	if balance == nil || cap_.Cmp(balance) <= 0 {
		return nil
	}
	return new(big.Int).Sub(cap_, balance)
}

func allEmpty(ladders [2][]ladder.Point) bool {
	return len(ladders[0]) == 0 && len(ladders[1]) == 0
}

func balStr(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}
