package tidefiprop

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
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

	// probe reports which pairs quote in at least one direction; a seam for tests.
	probe func(ctx context.Context, pairs [][2]string) ([]bool, error)
}

// Metadata persists the set of pool addresses already returned, so a
// growing asset list (new TideFi listings) only yields newly-added pairs
// on later calls instead of re-emitting everything each cycle. Pairs that
// did not quote are not recorded: they are re-probed on every call, since
// liquidity can appear later (e.g. RWA pairs when markets open).
type Metadata struct {
	Seen map[string]bool `json:"seen"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	u := &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
	u.probe = u.probeQuotes
	return u
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
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

	var candidates [][2]string
	for i := range tokens {
		for j := i + 1; j < len(tokens); j++ {
			t0, t1 := tokens[i], tokens[j]
			if t0 > t1 {
				t0, t1 = t1, t0
			}
			if !metadata.Seen[poolAddress(t0, t1)] {
				candidates = append(candidates, [2]string{t0, t1})
			}
		}
	}

	var quotes []bool
	if len(candidates) > 0 {
		if quotes, err = u.probe(ctx, candidates); err != nil {
			return nil, metadataBytes, err
		}
	}

	now := time.Now().Unix()
	pools := make([]entity.Pool, 0, len(candidates))
	for i, c := range candidates {
		if !quotes[i] {
			continue
		}
		addr := poolAddress(c[0], c[1])
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

// probeQuotes quotes one whole token (10^decimals) of tokenIn for both
// directions of every pair, and flags a pair if either direction returns a
// non-zero amount. A reverted call or failed decimals() read counts as no quote.
// A pair may quote one way only; the tracker zeroes the unquoted side.
func (u *PoolsListUpdater) probeQuotes(ctx context.Context, pairs [][2]string) ([]bool, error) {
	decimals := make(map[string]*uint8)
	var tokens []string
	decReq := u.ethrpcClient.NewRequest().SetContext(ctx)
	for _, p := range pairs {
		for _, tok := range p {
			if _, ok := decimals[tok]; ok {
				continue
			}
			var d uint8
			decimals[tok] = &d
			tokens = append(tokens, tok)
			decReq.AddCall(&ethrpc.Call{ABI: erc20ABI, Target: tok, Method: "decimals"}, []any{&d})
		}
	}
	decResp, err := decReq.TryAggregate()
	if err != nil {
		return nil, fmt.Errorf("tidefi-prop: probe decimals: %w", err)
	}
	failed := make(map[string]bool)
	for i, tok := range tokens {
		failed[tok] = !decResp.Result[i]
	}

	results := make([][2]quoteResult, len(pairs))
	calls := 0
	quoteReq := u.ethrpcClient.NewRequest().SetContext(ctx)
	var amount big.Int
	for i, p := range pairs {
		for dir := range p {
			tokIn, tokOut := p[dir], p[1-dir]
			if failed[tokIn] || failed[tokOut] {
				continue
			}
			amount.Exp(big.NewInt(10), big.NewInt(int64(*decimals[tokIn])), nil)
			quoteReq.AddCall(&ethrpc.Call{
				ABI:    swapperABI,
				Target: u.cfg.Address,
				Method: "quote",
				Params: []any{common.HexToAddress(tokIn), common.HexToAddress(tokOut),
					new(big.Int).Set(&amount), noLimitPrice, big.NewInt(swapFeePpm)},
			}, []any{&results[i][dir]})
			calls++
		}
	}
	if calls == 0 {
		return make([]bool, len(pairs)), nil
	}
	if _, err := quoteReq.TryAggregate(); err != nil {
		return nil, fmt.Errorf("tidefi-prop: probe quotes: %w", err)
	}

	quotes := make([]bool, len(pairs))
	for i := range results {
		for _, r := range results[i] {
			if r.AmountIn != nil && r.AmountOut != nil && r.AmountIn.Sign() > 0 && r.AmountOut.Sign() > 0 {
				quotes[i] = true
			}
		}
	}
	return quotes, nil
}
