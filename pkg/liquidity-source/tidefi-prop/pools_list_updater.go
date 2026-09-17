package tidefiprop

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
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
// tradeable token set. Every asset pair returned is assumed tradeable; the
// tracker's sampled quotes naturally return an empty ladder for any pair
// TideFi doesn't actually support.
type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

// Metadata persists the set of pool addresses already returned, so a
// growing asset list (new TideFi listings) only yields newly-added pairs
// on later calls instead of re-emitting everything each cycle.
type Metadata struct {
	Seen map[string]bool `json:"seen"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
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

	tokens, err := u.fetchAssets(ctx)
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

	now := time.Now().Unix()
	pools := make([]entity.Pool, 0, len(tokens))
	for i := range tokens {
		for j := i + 1; j < len(tokens); j++ {
			t0, t1 := tokens[i], tokens[j]
			if t0 > t1 {
				t0, t1 = t1, t0
			}

			addr := poolAddress(t0, t1)
			if metadata.Seen[addr] {
				continue
			}
			metadata.Seen[addr] = true

			pools = append(pools, entity.Pool{
				Address:   addr,
				Exchange:  u.cfg.DexID,
				Type:      DexType,
				Timestamp: now,
				Reserves:  entity.PoolReserves{"0", "0"},
				Tokens: []*entity.PoolToken{
					{Address: t0, Swappable: true},
					{Address: t1, Swappable: true},
				},
				Extra:       "{}",
				StaticExtra: string(staticExtraBytes),
			})
		}
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
