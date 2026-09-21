package everlongflamm

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Live checks against Base (skipped unless BASE_RPC_URL is set): lister -> tracker -> simulator at the head, with
// the simulator's venues compared to the deployed pool's previews at the tracked block, and the one settled swap
// replayed from a refresh pinned to its parent block.

func baseRPC(t *testing.T) string {
	t.Helper()
	u := os.Getenv("BASE_RPC_URL")
	if u == "" {
		t.Skip("BASE_RPC_URL not set")
	}
	return u
}

// retry429 retries a public endpoint's rate-limit answers (the refresh itself never retries).
type retry429 struct{}

func (retry429) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	for attempt := 0; ; attempt++ {
		r := req.Clone(req.Context())
		r.Body = io.NopCloser(bytes.NewReader(body))
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests || attempt == 12 {
			return resp, err
		}
		_ = resp.Body.Close()
		time.Sleep(time.Duration(250*(attempt+1)) * time.Millisecond)
	}
}

func liveClient(t *testing.T, url string) *ethrpc.Client {
	t.Helper()
	rc, err := rpc.DialOptions(context.Background(), url, rpc.WithHTTPClient(&http.Client{Transport: retry429{}}))
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return ethrpc.NewWithClient(ethclient.NewClient(rc)).SetMulticallContract(multicall3)
}

// baseConfig is the production configuration of the Base listing: every margin at its default.
func baseConfig() *Config {
	return &Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: c104.Factory.Hex()}
}

// tapeConfig is baseConfig restricted to the pool the recorded tapes answer for. The tapes are a recording of one
// pool's listing round, so a second registered pool of the same deployment would send that pool's reads into a tape
// that has no answer for them: the offline replays name the pool they are about, as the fork suites that list
// against a chain do (fork_multipool_test.go list). README, "Adding a pool whose hooks are of a registered kind".
func tapeConfig() *Config {
	cfg := baseConfig()
	cfg.Pools = []string{c104.Pool.Hex()}
	return cfg
}

// parityConfig is baseConfig with the policy's margins stamped as set, zero included (off): the parity tests compare
// every quote with the chain, band edges and clock deadlines included, where a default margin would refuse.
func parityConfig(policy Policy) *Config {
	cfg := baseConfig()
	cfg.LeverRouting, cfg.QuoteDonatedVenues = policy.LeverRouting, policy.QuoteDonatedVenues
	cfg.LeverMinEdgeBps = &policy.LeverMinEdgeBps
	cfg.PriceBandMarginBps, cfg.PriceAgeMarginSec = &policy.PriceBandMarginBps, &policy.PriceAgeMarginSec
	cfg.SpreadAgeMarginSec, cfg.DebtDriftSec = &policy.SpreadAgeMarginSec, &policy.DebtDriftSec
	cfg.MaxSnapshotAgeSec = &policy.MaxSnapshotAgeSec
	return cfg
}

func liveList(t *testing.T, ctx context.Context, client *ethrpc.Client, cfg *Config) entity.Pool {
	t.Helper()
	pools, md, err := NewPoolsListUpdater(cfg, client).GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, lowerHex(c104.Pool), pools[0].Address)
	// The cursor suppresses an unchanged relisting.
	again, _, err := NewPoolsListUpdater(cfg, client).GetNewPools(ctx, md)
	require.NoError(t, err)
	require.Empty(t, again)
	return pools[0]
}

// simAt builds a simulator from a tracked entity quoting at the snapshot timestamp with margins off.
func simAt(t *testing.T, tracked entity.Pool) (*PoolSimulator, *Extra) {
	t.Helper()
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "attest failure %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)
	ts := sim.state.Timestamp
	sim.nowFn = func() uint64 { return ts }
	return sim, &extra
}

// settlementOnly are the pool's own errors that only an executed fill can hit -- previewSwap and previewLever plan
// against a view, the fill settles -- so the chain's preview can accept a size the port refuses with one of them:
// the Router legs through Morpho and the re-asserted gates. Each occurrence is confirmed against the settlement
// itself (confirmSettlement).
var settlementOnly = []error{ErrInsufficientLiquidity, ErrInsufficientCollateral, ErrRateCeiling, ErrUnhealthy,
	ErrOracleBand, ErrDebtCapExceeded, ErrSupplyCapExceeded, ErrLedgerBoundBreached, ErrExitWorsensLedger,
	errMMInsufficientLiquidity, errMMInsufficientCollateral}

// policyOnly are the port's own declared refusals: once the keeper's post has lapsed the port quotes neither
// direction of the leverage venue (pool_simulator.go settleLever), where the pool reverts a lever-up with
// SpreadUnavailable and plans a lever-down at the stored degrade value (FLAMMLeverLib._spread, :158-166), a fill
// this port will not quote, which the pool then fills or refuses on its own gates. They are counted apart, by
// whether the chain reverts too or fills (the fork suite's judgeFill does the same), and the fills are not
// confirmed against the settlement, which would fill. A spread refusal while the chain's post answers at the
// block (chainSpreadLive) is a mismatch, whatever the chain answers.
var policyOnly = []error{ErrSpreadNotLive}

// policyRefusal reports whether simErr is a declared policy refusal the chain's state at the block supports: a
// spread refusal while the chain's post does not answer.
func policyRefusal(simErr error, spreadLive bool) bool {
	return isOneOf(simErr, policyOnly) && !spreadLive
}

func isOneOf(err error, list []error) bool {
	for _, e := range list {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

func isSettlementOnly(err error) bool { return isOneOf(err, settlementOnly) }

// liveTally separates what a live comparison can produce. They are not the same evidence: a matched revert is the
// chain refusing too, while a settlement-only refusal is the port refusing a fill the pool's own preview planned,
// and only the settlement itself can say whether the pool would have filled it -- so each one is confirmed with an
// eth_call of that settlement (confirmSettlement) and counted apart. Folding the two into one "refused alike"
// number is what lets a port that refuses everything with a settlement-only error pass the live suite. A declared
// policy refusal is counted by whether the chain reverts for the same cause (policyAgrees) or fills.
type liveTally struct {
	identical, chainReverted, settlementOnly, policyRefusedRevert, policyRefusedFill int
}

func (l *liveTally) String() string {
	return fmt.Sprintf("%d identical, %d refused alike, %d settlement-only refusals confirmed against the pool, "+
		"%d declared policy refusals the chain reverts too, %d declared policy refusals the chain fills",
		l.identical, l.chainReverted, l.settlementOnly, l.policyRefusedRevert, l.policyRefusedFill)
}

// compareLive checks one venue quote against the pool's preview at the tracked block; spreadLive is whether the
// chain's spread post answers there (chainSpreadLive), which decides whether a spread refusal is policy or a
// mismatch.
func compareLive(t *testing.T, ctx context.Context, client *ethrpc.Client, block *big.Int, method string, dir bool,
	amount uint64, f *fill, simErr error, spreadLive bool, tally *liveTally) (ok bool) {
	t.Helper()
	pool := c104.Pool
	data, err := flammABI.Pack(method, dir, new(big.Int).SetUint64(amount))
	require.NoError(t, err)
	ret, callErr := client.GetETHClient().CallContract(ctx, ethereum.CallMsg{To: &pool, Data: data}, block)
	if callErr != nil {
		revert, isRevert := revertData(callErr)
		require.True(t, isRevert, "%s(%v,%d): %v", method, dir, amount, callErr)
		want := revertError(revert)
		require.NotNil(t, want, "%s(%v,%d): unmapped revert %x", method, dir, amount, revert)
		if policyRefusal(simErr, spreadLive) {
			t.Logf("%s(%v,%d): declared policy refusal %v, chain reverted %v", method, dir, amount, simErr, want)
			tally.policyRefusedRevert++
			return false
		}
		require.ErrorIs(t, simErr, want, "%s(%v,%d): chain reverted %v", method, dir, amount, want)
		tally.chainReverted++
		return false
	}
	vals, err := flammABI.Methods[method].Outputs.Unpack(ret)
	require.NoError(t, err)
	if simErr != nil {
		if isOneOf(simErr, policyOnly) {
			require.True(t, policyRefusal(simErr, spreadLive), "%s(%v,%d): spread refusal %v while the chain's post "+
				"answers, chain %v", method, dir, amount, simErr, vals)
			t.Logf("%s(%v,%d): declared policy refusal %v, chain fills", method, dir, amount, simErr)
			tally.policyRefusedFill++
			return false
		}
		require.True(t, isSettlementOnly(simErr), "%s(%v,%d): chain %v, sim refused %v", method, dir, amount, vals,
			simErr)
		confirmSettlement(t, ctx, client, block, method, dir, amount, simErr)
		tally.settlementOnly++
		return false
	}
	used, _ := wordOf(vals[0])
	out, _ := wordOf(vals[1])
	require.Equal(t, used.Dec(), f.used.Dec(), "%s(%v,%d) used", method, dir, amount)
	require.Equal(t, out.Dec(), f.out.Dec(), "%s(%v,%d) out", method, dir, amount)
	tally.identical++
	return true
}

// settlementABI is the pool's exact-input entrypoints, the calls an executor makes (PR A's IFLAMM). They are not in
// the embedded ABI, which holds only the views the lister and tracker call.
var settlementABI = func() gethabi.ABI {
	a, err := gethabi.JSON(strings.NewReader(`[
 {"type":"function","name":"swap","stateMutability":"nonpayable","inputs":[{"name":"tokenIn","type":"address"},{"name":"tokenOut","type":"address"},{"name":"amountIn","type":"uint256"},{"name":"minAmountOut","type":"uint256"},{"name":"to","type":"address"},{"name":"deadline","type":"uint256"}],"outputs":[{"name":"amountInUsed","type":"uint256"},{"name":"amountOut","type":"uint256"}]},
 {"type":"function","name":"leverUp","stateMutability":"nonpayable","inputs":[{"name":"poolAssetIn","type":"uint256"},{"name":"minLoanOut","type":"uint256"},{"name":"to","type":"address"},{"name":"deadline","type":"uint256"}],"outputs":[{"name":"amountInUsed","type":"uint256"},{"name":"loanOut","type":"uint256"}]},
 {"type":"function","name":"leverDown","stateMutability":"nonpayable","inputs":[{"name":"loanIn","type":"uint256"},{"name":"minPoolOut","type":"uint256"},{"name":"to","type":"address"},{"name":"deadline","type":"uint256"}],"outputs":[{"name":"amountInUsed","type":"uint256"},{"name":"poolOut","type":"uint256"}]},
 {"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"who","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}
]`))
	if err != nil {
		panic(err)
	}
	return a
}()

// liveTaker is the account the confirmation settles from: it holds nothing on chain and is funded by overrides.
var liveTaker = common.HexToAddress("0x00000000000000000000000000000000000000f1")

// fiatTokenAllowanceSlot is FiatTokenV2_2's `allowed` mapping, beside the balances at fiatTokenBalanceSlot
// (fork_harness_test.go). Both cbBTC and USDC on Base use this layout; the balance override is checked against
// balanceOf before the settlement runs, so a wrong slot fails the test instead of confirming nothing.
const fiatTokenAllowanceSlot = 10

func mapSlot(key common.Address, slot int64) common.Hash {
	var buf [64]byte
	copy(buf[12:32], key[:])
	copy(buf[32:], common.BigToHash(big.NewInt(slot)).Bytes())
	return crypto.Keccak256Hash(buf[:])
}

// takerOverrides funds liveTaker with amount of token and approves the pool for it.
func takerOverrides(t *testing.T, ctx context.Context, client *ethrpc.Client, block *big.Int, token common.Address,
	amount *big.Int) map[common.Address]gethclient.OverrideAccount {
	t.Helper()
	balance := balanceSlot(liveTaker)
	// allowed[owner][spender]: keccak(spender . keccak(owner . slot)).
	inner := mapSlot(liveTaker, fiatTokenAllowanceSlot)
	allow := crypto.Keccak256Hash(append(common.LeftPadBytes(c104.Pool[:], 32), inner[:]...))
	value := common.BigToHash(amount)
	out := map[common.Address]gethclient.OverrideAccount{
		token: {StateDiff: map[common.Hash]common.Hash{balance: value, allow: value}},
	}
	// The slot is the one the token really uses, or the confirmation would prove nothing.
	data, err := settlementABI.Pack("balanceOf", liveTaker)
	require.NoError(t, err)
	ret, err := gethclient.New(client.GetETHClient().Client()).CallContract(ctx,
		ethereum.CallMsg{To: &token, Data: data}, block, &out)
	require.NoError(t, err)
	require.Equal(t, amount.String(), new(big.Int).SetBytes(ret).String(), "balance slot of %s", token)
	return out
}

// confirmSettlement runs the settlement the refused preview belongs to as an eth_call at the same block, from an
// account the overrides fund and approve, and requires the pool to refuse it too. A settlement-only refusal is the
// port saying the fill the preview planned does not survive the Router legs; if the pool fills it, the port is
// wrong, and nothing else in the live suite would notice.
func confirmSettlement(t *testing.T, ctx context.Context, client *ethrpc.Client, block *big.Int, method string,
	dir bool, amount uint64, simErr error) {
	t.Helper()
	prof := &c104
	where := fmt.Sprintf("%s(%v,%d) refused %v", method, dir, amount, simErr)
	in := new(big.Int).SetUint64(amount)
	deadline := new(big.Int).SetUint64(1 << 40)
	var call string
	var args []any
	token := prof.PoolAsset
	switch {
	case method == "previewSwap" && dir:
		call, args = "swap", []any{prof.PoolAsset, prof.LoanAsset, in, big.NewInt(1), liveTaker, deadline}
	case method == "previewSwap":
		call, args, token = "swap", []any{prof.LoanAsset, prof.PoolAsset, in, big.NewInt(1), liveTaker, deadline},
			prof.LoanAsset
	case dir:
		call, args = "leverUp", []any{in, big.NewInt(1), liveTaker, deadline}
	default:
		call, args, token = "leverDown", []any{in, big.NewInt(1), liveTaker, deadline}, prof.LoanAsset
	}
	data, err := settlementABI.Pack(call, args...)
	require.NoError(t, err)
	overrides := takerOverrides(t, ctx, client, block, token, in)
	pool := prof.Pool
	ret, callErr := gethclient.New(client.GetETHClient().Client()).CallContract(ctx,
		ethereum.CallMsg{From: liveTaker, To: &pool, Data: data}, block, &overrides)
	require.Error(t, callErr, "%s: the pool settled it (%x)", where, ret)
	revert, isRevert := revertData(callErr)
	require.True(t, isRevert, "%s: %v", where, callErr)
	want := forkRevertError(revert)
	require.NotNil(t, want, "%s: unmapped settlement revert %x", where, revert)
	t.Logf("settlement-only refusal confirmed: %s, the pool reverts %v", where, want)
}

// revertData extracts eth_call revert data from an RPC error.
func revertData(err error) ([]byte, bool) {
	var de interface{ ErrorData() any }
	if !errors.As(err, &de) {
		return nil, false
	}
	s, ok := de.ErrorData().(string)
	if !ok {
		return []byte{}, true // execution reverted without data
	}
	return common.FromHex(s), true
}

// liveGrid is n log-spaced amounts from lo to hi.
func liveGrid(lo, hi uint64, n int) []uint64 {
	out := make([]uint64, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, uint64(float64(lo)*math.Pow(float64(hi)/float64(lo), float64(i)/float64(n-1))))
	}
	return out
}

func TestLiveHead(t *testing.T) {
	url := baseRPC(t)
	ctx := context.Background()
	client := liveClient(t, url)
	cfg := parityConfig(Policy{})
	listed := liveList(t, ctx, client, cfg)
	tracked, err := NewPoolTracker(cfg, client).GetNewPoolState(ctx, listed, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, extra := simAt(t, tracked)
	block := new(big.Int).SetUint64(tracked.BlockNumber)
	t.Logf("head %d ts %d: probes %d, reserves %v, paused %v levPaused %v spread %s", tracked.BlockNumber,
		sim.state.Timestamp, extra.Probes, tracked.Reserves, sim.state.Paused, sim.state.LevPaused,
		sim.state.Hooks.Spread.EverlongSpread.Spread.Dec())

	// The dependency set resolves against the chain: four Chainlink aggregators behind the proxies the wiring
	// names, the swap hook, the spread hook and the Router (pool_tracker.go GetDependencies).
	deps, stored, err := NewPoolTracker(cfg, client).GetDependencies(ctx, tracked)
	require.NoError(t, err)
	require.False(t, stored)
	require.Len(t, deps, 7)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(tracked.StaticExtra), &se))
	for _, proxy := range []common.Address{se.Aggregators[0], se.Aggregators[1], se.SequencerFeed} {
		require.NotContains(t, deps, lowerHex(proxy), "the proxy, not the aggregator that emits its rounds")
	}
	for _, a := range []common.Address{se.Hooks[0], se.Hooks[5], se.Router} {
		require.Contains(t, deps, lowerHex(a))
	}
	t.Logf("live head dependencies: %v", deps)

	now := sim.state.Timestamp
	spreadLive := chainSpreadLive(sim, now)
	var tally liveTally
	for _, dir := range []struct {
		sell   bool
		lo, hi uint64
	}{{true, 1, 10_000_000}, {false, 100, 10_000_000_000}} {
		for _, a := range liveGrid(dir.lo, dir.hi, 24) {
			f, err := sim.quoteSwap(dir.sell, uint256.NewInt(a), now)
			compareLive(t, ctx, client, block, "previewSwap", dir.sell, a, f, err, spreadLive, &tally)
		}
	}
	for _, dir := range []struct {
		up     bool
		lo, hi uint64
	}{{true, 1, 10_000_000}, {false, 100, 10_000_000_000}} {
		for _, a := range liveGrid(dir.lo, dir.hi, 24) {
			f, err := sim.quoteLever(dir.up, uint256.NewInt(a), now)
			compareLive(t, ctx, client, block, "previewLever", dir.up, a, f, err, spreadLive, &tally)
		}
	}
	t.Logf("live head (spread post live %v): %s", spreadLive, tally.String())
	require.Positive(t, tally.identical)
}

func TestLiveRealSell(t *testing.T) {
	url := baseRPC(t)
	ctx := context.Background()
	client := liveClient(t, url)
	cfg := baseConfig()
	listed := liveList(t, ctx, client, cfg)
	tracked, err := NewPoolTracker(cfg, client).GetNewPoolStateAtBlock(ctx, listed, big.NewInt(51302915))
	require.NoError(t, err)
	require.Equal(t, uint64(51302915), tracked.BlockNumber)
	sim, _ := simAt(t, tracked)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(15000)},
		TokenOut:      sim.Info.Tokens[1]})
	require.NoError(t, err)
	require.Equal(t, "11301759", res.TokenAmountOut.Amount.String(), "tx 0x46c3cd72... paid 11301759 USDC")
	require.Equal(t, "0", res.RemainingTokenAmountIn.Amount.String())
}

// TestLiveSettlement proves the confirmation a settlement-only refusal goes through can tell the two answers
// apart on the deployed pool: the settlement of the one real swap fills for exactly what it paid on chain, and a
// settlement the pool refuses reverts with the pool's own error. Without this, confirmSettlement could be
// confirming nothing (an override that does not fund the taker makes every settlement revert).
func TestLiveSettlement(t *testing.T) {
	url := baseRPC(t)
	ctx := context.Background()
	client := liveClient(t, url)
	prof := &c104
	block := big.NewInt(51302915)
	deadline := new(big.Int).SetUint64(1 << 40)

	fill, err := settlementABI.Pack("swap", prof.PoolAsset, prof.LoanAsset, big.NewInt(15_000), big.NewInt(1),
		liveTaker, deadline)
	require.NoError(t, err)
	overrides := takerOverrides(t, ctx, client, block, prof.PoolAsset, big.NewInt(15_000))
	pool := prof.Pool
	ret, err := gethclient.New(client.GetETHClient().Client()).CallContract(ctx,
		ethereum.CallMsg{From: liveTaker, To: &pool, Data: fill}, block, &overrides)
	require.NoError(t, err, "the settlement of tx 0x46c3cd72... runs")
	vals, err := settlementABI.Methods["swap"].Outputs.Unpack(ret)
	require.NoError(t, err)
	require.Equal(t, "15000", vals[0].(*big.Int).String())
	require.Equal(t, "11301759", vals[1].(*big.Int).String(), "tx 0x46c3cd72... paid 11301759 USDC")

	// And a size the pool's band refuses settles as a revert, not as a fill: confirmSettlement requires one.
	confirmSettlement(t, ctx, client, block, "previewSwap", true, 100_000_000, ErrPriceBand)
}

// Batch limits and historical blocks (skipped unless BASE_RPC_URL is set). mainnet.base.org answers a JSON-RPC
// batch of more than ten calls with a single error object; TestLiveBatchLimit records what the tracker does with
// the endpoint as is, and TestLiveBlocks goes through a transport that splits batches into tens (and retries 429s)
// so the refresh can be checked at historical blocks.

type splitBatch struct{ max int }

func (s splitBatch) do(req *http.Request, body []byte) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		r := req.Clone(req.Context())
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests || attempt == 20 {
			return resp, err
		}
		_ = resp.Body.Close()
		time.Sleep(time.Duration(300*(attempt+1)) * time.Millisecond)
	}
}

func (s splitBatch) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var msgs []stdjson.RawMessage
	if s.max <= 0 || len(bytes.TrimSpace(body)) == 0 || bytes.TrimSpace(body)[0] != '[' ||
		stdjson.Unmarshal(body, &msgs) != nil || len(msgs) <= s.max {
		return s.do(req, body)
	}
	var merged []stdjson.RawMessage
	var last *http.Response
	for i := 0; i < len(msgs); i += s.max {
		chunk, _ := stdjson.Marshal(msgs[i:min(i+s.max, len(msgs))])
		resp, err := s.do(req, chunk)
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var part []stdjson.RawMessage
		if err := stdjson.Unmarshal(raw, &part); err != nil {
			return nil, fmt.Errorf("chunk answer %s: %w", raw, err)
		}
		merged = append(merged, part...)
		last = resp
	}
	out, _ := stdjson.Marshal(merged)
	last.Body = io.NopCloser(bytes.NewReader(out))
	last.ContentLength = int64(len(out))
	last.Header.Del("Content-Length")
	return last, nil
}

func splitBatchClient(t *testing.T, url string, maxBatch int) *ethrpc.Client {
	t.Helper()
	rc, err := rpc.DialOptions(context.Background(), url,
		rpc.WithHTTPClient(&http.Client{Transport: splitBatch{max: maxBatch}}))
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return ethrpc.NewWithClient(ethclient.NewClient(rc)).SetMulticallContract(multicall3)
}

// TestLiveBatchLimit: the lister and the tracker against the endpoint as is (no batch splitting in the
// transport): both chunk their own batches (multicall.go batchCall), so both succeed.
func TestLiveBatchLimit(t *testing.T) {
	url := baseRPC(t)
	ctx := context.Background()
	client := splitBatchClient(t, url, 0)
	cfg := baseConfig()
	pools, _, err := NewPoolsListUpdater(cfg, client).GetNewPools(ctx, nil)
	require.NoError(t, err, "lister without splitting")
	require.Len(t, pools, 1)
	tracked, err := NewPoolTracker(cfg, client).GetNewPoolState(ctx, pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err, "tracker without splitting")
	var extra Extra
	require.NoError(t, stdjson.Unmarshal([]byte(tracked.Extra), &extra))
	t.Logf("tracker without splitting at %d: attested %v, %d probes", tracked.BlockNumber, extra.Attested, extra.Probes)
	require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
}

// TestLiveBlocks refreshes at six historical blocks and compares both venues
// with the pool's previews at each block over dense grids plus unit windows at the swap band edges.
func TestLiveBlocks(t *testing.T) {
	url := baseRPC(t)
	ctx := context.Background()
	client := splitBatchClient(t, url, 10)
	cfg := parityConfig(Policy{})
	listed := liveList(t, ctx, client, cfg)
	for _, b := range []uint64{51298500, 51302915, 51310000, 51318000, 51326000, 51333700} {
		tracked, err := NewPoolTracker(cfg, client).GetNewPoolStateAtBlock(ctx, listed, new(big.Int).SetUint64(b))
		require.NoError(t, err, "block %d", b)
		sim, extra := simAt(t, tracked)
		block := new(big.Int).SetUint64(b)
		now := sim.state.Timestamp
		spreadLive := chainSpreadLive(sim, now)
		var tally liveTally
		for _, dir := range []struct {
			sell   bool
			lo, hi uint64
			grid   []uint64
		}{{true, 1, 3_000_000, probeSellGrid}, {false, 100, 3_000_000_000, probeBuyGrid}} {
			amounts := liveGrid(dir.lo, dir.hi, 20)
			if lo, _, ok := edge(dir.grid, func(a *uint256.Int) bool {
				_, err := sim.quoteSwap(dir.sell, a, now)
				return err == nil
			}); ok {
				c := lo.Uint64()
				for a := c - min(c-1, 4); a <= c+4; a++ {
					amounts = append(amounts, a)
				}
			}
			for _, a := range amounts {
				f, err := sim.quoteSwap(dir.sell, uint256.NewInt(a), now)
				compareLive(t, ctx, client, block, "previewSwap", dir.sell, a, f, err, spreadLive, &tally)
			}
			for _, a := range liveGrid(dir.lo, dir.hi, 8) {
				f, err := sim.quoteLever(dir.sell, uint256.NewInt(a), now)
				compareLive(t, ctx, client, block, "previewLever", dir.sell, a, f, err, spreadLive, &tally)
			}
		}
		t.Logf("block %d ts %d: probes %d, paused %v levPaused %v spread post live %v, reserves %v: %s", b, now,
			extra.Probes, sim.state.Paused, sim.state.LevPaused, spreadLive, tracked.Reserves, tally.String())
	}
}
