package tidefiprop

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdater discovers pools from the Taker API's tidefi_markets
// message (pushed immediately on connect -- see docs.ulmomarkets.com's
// Taker API section): TideFi has no on-chain enumerable asset registry
// (pricing config is pushed by a trusted off-chain signer with no
// corresponding event log), so this is the only source of truth for the
// tradeable token set. The payload carries tokens only (a swap crosses two
// per-token books quoted in USD), so a pair is listed only if an on-chain
// quote probe returns a non-zero amount in at least one direction. The lens
// getOrderBook is per-token and needs the engine address (not on-chain
// discoverable); every registered token has a book even when no pair quotes,
// so it cannot decide pairs and is not used.
type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	// Seams for tests. probe reports which pairs quote in at least one
	// direction; readDecimals returns 0 for a failed read; callQuotes reports
	// whether each quote returned a non-zero amount.
	probe        func(ctx context.Context, pairs [][2]string) ([]bool, error)
	readDecimals func(ctx context.Context, tokens []string) ([]uint8, error)
	callQuotes   func(ctx context.Context, calls []quoteCall) ([]bool, error)

	// In-memory only: losing it on restart costs one extra probe round, never
	// correctness (Seen, the only persisted state, decides what is listed).
	mu       sync.Mutex
	run      int                // list-run counter
	decimals map[string]uint8   // token -> decimals; only successful non-zero reads
	backoff  map[string]backoff // pair address -> retry schedule of a non-quoting pair
	failed   map[string]int     // token -> non-quoting probes since it last quoted
}

// List runs tick every 10s (pool-service DefaultRunInterval, sequential), so a
// run must stay well below that. A full 100-token sweep (4950 pairs) takes 25 runs.
const (
	probeChunk  = 50  // pairs per quote multicall (100 quote calls)
	probeBudget = 200 // pairs probed per run (4 multicalls)

	backoffBase = 30  // runs (~5 min) before a non-quoting pair is re-probed
	backoffMax  = 360 // runs (~1 h) cap; the delay doubles up to here

	deadAfter       = 10 // non-quoting probes after which a token with no listed pair is dead
	deadSampleEvery = 60 // runs (~10 min): only then are pairs of a dead token probed
)

type backoff struct{ next, delay int }

type quoteCall struct {
	in, out string
	dec     uint8
}

// Metadata persists the set of pool addresses already returned, so a
// growing asset list (new TideFi listings) only yields newly-added pairs
// on later calls instead of re-emitting everything each cycle. Pairs that
// did not quote are not recorded: they are re-probed with backoff (see
// GetNewPools), since liquidity can appear later (e.g. RWA pairs when markets open).
type Metadata struct {
	Seen map[string]bool `json:"seen"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	u := &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient,
		decimals: map[string]uint8{}, backoff: map[string]backoff{}, failed: map[string]int{}}
	u.probe = u.probeQuotes
	u.readDecimals = u.readDecimalsRPC
	u.callQuotes = u.callQuotesRPC
	return u
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.run++

	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}
	if metadata.Seen == nil {
		metadata.Seen = make(map[string]bool)
	}

	tokens, err := u.getTokens(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		Address: strings.ToLower(u.cfg.Address),
		Vault:   strings.ToLower(u.cfg.Vault),
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	candidates := u.pickCandidates(tokens, metadata.Seen)

	// Probe in chunks; state is updated only after every chunk succeeded.
	quotes := make([]bool, 0, len(candidates))
	for lo := 0; lo < len(candidates); lo += probeChunk {
		q, err := u.probe(ctx, candidates[lo:min(lo+probeChunk, len(candidates))])
		if err != nil {
			return nil, metadataBytes, err
		}
		quotes = append(quotes, q...)
	}

	now := time.Now().Unix()
	pools := make([]entity.Pool, 0, len(candidates))
	for i, c := range candidates {
		addr := poolAddress(c[0], c[1])
		if !quotes[i] {
			b := backoff{delay: backoffBase}
			if old, ok := u.backoff[addr]; ok {
				b.delay = min(old.delay*2, backoffMax)
			}
			b.next = u.run + b.delay
			u.backoff[addr] = b
			u.failed[c[0]]++
			u.failed[c[1]]++
			continue
		}
		delete(u.backoff, addr)
		delete(u.failed, c[0])
		delete(u.failed, c[1])
		metadata.Seen[addr] = true

		pools = append(pools, entity.Pool{
			Address:   addr,
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: c[0], Swappable: true},
				{Address: c[1], Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

// takerMarketsMsg is the JSON-RPC-shaped message pushed once, immediately
// on connect: {"method": "tidefi_markets", "params": {...}}.
type takerMarketsMsg struct {
	Method string       `json:"method"`
	Params takerMarkets `json:"params"`
}

type takerMarkets struct {
	Assets []struct {
		TokenAddress string `json:"token_address"`
	} `json:"assets"`
}

// getTokens returns the configured static token list if set, else the
// Taker API's asset list.
func (u *PoolsListUpdater) getTokens(ctx context.Context) ([]string, error) {
	if len(u.cfg.Tokens) == 0 {
		return u.fetchAssets(ctx)
	}
	tokens := make([]string, len(u.cfg.Tokens))
	for i, t := range u.cfg.Tokens {
		tokens[i] = strings.ToLower(t)
	}
	return tokens, nil
}

// fetchAssets connects to the Taker API, reads the first (and only, for our
// purposes) pushed message, and returns its quotable token addresses.
func (u *PoolsListUpdater) fetchAssets(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, takerAPITimeout)
	defer cancel()

	header := http.Header{"Authorization": {"Bearer " + u.cfg.AuthToken}}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.cfg.TakerAPIURL, header)
	if err != nil {
		return nil, fmt.Errorf("tidefi-prop: dial taker API: %w", err)
	}
	defer func(conn *websocket.Conn) {
		_ = conn.Close()
	}(conn)

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	}

	_, data, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("tidefi-prop: read taker API: %w", err)
	}

	var msg takerMarketsMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("tidefi-prop: unmarshal taker API message: %w", err)
	}
	if msg.Method != "tidefi_markets" {
		return nil, fmt.Errorf("tidefi-prop: expected tidefi_markets, got %q", msg.Method)
	}

	tokens := make([]string, 0, len(msg.Params.Assets))
	for _, a := range msg.Params.Assets {
		tokens = append(tokens, strings.ToLower(a.TokenAddress))
	}
	return tokens, nil
}

// poolAddress is synthetic, following manta-prop's pattern: TideFi has no
// per-pair contract of its own (it's a single fixed swapper backing every
// pair), so a fixed namespace plus the pair already disambiguates every
// pool without needing the swapper address itself. token0/token1 must
// already be sorted -- the Taker API doesn't guarantee asset order, and an
// unsorted pair would mint duplicate pools for the same tokens.
func poolAddress(token0, token1 string) string {
	return "tidefiprop_" + token0 + "_" + token1
}

// pickCandidates returns up to probeBudget unseen pairs to probe this run:
// never-probed pairs first (new tokens get priority), then due backoff pairs,
// oldest due first, which rotates through them. Pairs of a dead token (never
// listed, failed against deadAfter partners) are skipped except on sample runs.
func (u *PoolsListUpdater) pickCandidates(tokens []string, seen map[string]bool) [][2]string {
	type cand struct {
		pair [2]string
		next int // 0 = never probed
	}
	var cands []cand
	alive := make(map[string]bool)
	for i := range tokens {
		for j := i + 1; j < len(tokens); j++ {
			t0, t1 := tokens[i], tokens[j]
			if t0 > t1 {
				t0, t1 = t1, t0
			}
			addr := poolAddress(t0, t1)
			if seen[addr] {
				alive[t0], alive[t1] = true, true
				continue
			}
			b, ok := u.backoff[addr]
			if ok && u.run < b.next {
				continue
			}
			cands = append(cands, cand{[2]string{t0, t1}, b.next})
		}
	}
	dead := func(t string) bool { return !alive[t] && u.failed[t] >= deadAfter }
	sample := u.run%deadSampleEvery == 0
	kept := cands[:0]
	for _, c := range cands {
		if sample || (!dead(c.pair[0]) && !dead(c.pair[1])) {
			kept = append(kept, c)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].next < kept[j].next })
	kept = kept[:min(len(kept), probeBudget)]
	pairs := make([][2]string, len(kept))
	for i, c := range kept {
		pairs[i] = c.pair
	}
	return pairs
}

// probeQuotes quotes one whole token (10^decimals) of tokenIn for both
// directions of every pair, and flags a pair if either direction returns a
// non-zero amount. A failed or zero decimals() read counts as no quote and is
// not cached; a reverted quote counts as no quote. A pair may quote one way
// only; the tracker zeroes the unquoted side. Decimals never change, so they
// are cached in memory per token (persisting them would only save one cheap
// read per token after a restart).
func (u *PoolsListUpdater) probeQuotes(ctx context.Context, pairs [][2]string) ([]bool, error) {
	var missing []string
	asked := make(map[string]bool)
	for _, p := range pairs {
		for _, tok := range p {
			if _, ok := u.decimals[tok]; !ok && !asked[tok] {
				asked[tok] = true
				missing = append(missing, tok)
			}
		}
	}
	if len(missing) > 0 {
		decs, err := u.readDecimals(ctx, missing)
		if err != nil {
			return nil, fmt.Errorf("tidefi-prop: probe decimals: %w", err)
		}
		for i, tok := range missing {
			if decs[i] > 0 {
				u.decimals[tok] = decs[i]
			}
		}
	}

	var calls []quoteCall
	var owner []int // calls[k] belongs to pairs[owner[k]]
	for i, p := range pairs {
		for dir := range p {
			tokIn, tokOut := p[dir], p[1-dir]
			decIn, okIn := u.decimals[tokIn]
			_, okOut := u.decimals[tokOut]
			if okIn && okOut {
				calls = append(calls, quoteCall{tokIn, tokOut, decIn})
				owner = append(owner, i)
			}
		}
	}
	quotes := make([]bool, len(pairs))
	if len(calls) == 0 {
		return quotes, nil
	}
	ok, err := u.callQuotes(ctx, calls)
	if err != nil {
		return nil, fmt.Errorf("tidefi-prop: probe quotes: %w", err)
	}
	for k, q := range ok {
		quotes[owner[k]] = quotes[owner[k]] || q
	}
	return quotes, nil
}

func (u *PoolsListUpdater) readDecimalsRPC(ctx context.Context, tokens []string) ([]uint8, error) {
	decs := make([]uint8, len(tokens))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, tok := range tokens {
		req.AddCall(&ethrpc.Call{ABI: erc20ABI, Target: tok, Method: "decimals"}, []any{&decs[i]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	for i := range decs {
		if !resp.Result[i] {
			decs[i] = 0
		}
	}
	return decs, nil
}

func (u *PoolsListUpdater) callQuotesRPC(ctx context.Context, calls []quoteCall) ([]bool, error) {
	results := make([]quoteResult, len(calls))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, c := range calls {
		amount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(c.dec)), nil)
		req.AddCall(&ethrpc.Call{
			ABI:    swapperABI,
			Target: u.cfg.Address,
			Method: "quote",
			Params: []any{common.HexToAddress(c.in), common.HexToAddress(c.out),
				amount, noLimitPrice, big.NewInt(swapFeePpm)},
		}, []any{&results[i]})
	}
	if _, err := req.TryAggregate(); err != nil {
		return nil, err
	}
	quotes := make([]bool, len(calls))
	for i, r := range results {
		quotes[i] = r.AmountIn != nil && r.AmountOut != nil && r.AmountIn.Sign() > 0 && r.AmountOut.Sign() > 0
	}
	return quotes, nil
}
