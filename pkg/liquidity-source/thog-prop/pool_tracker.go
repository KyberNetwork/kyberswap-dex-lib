package thogprop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, client *ethrpc.Client) (*PoolTracker, error) {
	if cfg == nil || cfg.BlockTimeMs < 0 || cfg.MaxSnapshotAgeMs < 0 || cfg.RequestTimeoutMs < 0 || cfg.RequestTimeoutMs > 60_000 {
		return nil, fmt.Errorf("thog-prop: invalid refresh/freshness config")
	}
	return &PoolTracker{config: cfg, ethrpcClient: client}, nil
}

func defaultMs(value, fallback int64) int64 {
	if value == 0 {
		return fallback
	}
	return value
}

type refreshError struct {
	kind, stage string
	err         error
}

func (e *refreshError) Error() string {
	return fmt.Sprintf("thog-prop: %s (%s): %v", e.stage, e.kind, e.err)
}
func (e *refreshError) Unwrap() error                    { return e.err }
func refreshFailure(kind, stage string, err error) error { return &refreshError{kind, stage, err} }
func classifyRefreshError(err error) string {
	var detailed *refreshError
	if errors.As(err, &detailed) {
		return detailed.kind
	}
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{"block not found", "header not found", "unknown block", "missing trie"} {
		if strings.Contains(message, fragment) {
			return "provider_lag"
		}
	}
	if errors.Is(err, ErrInvalidState) {
		return "validation"
	}
	return "transport"
}

// Retry the whole read, never an issuer batch at a different block. Each attempt
// has one deadline, including any capability fallback; at most 4.3 s by default.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return p, err
		}
		started := time.Now()
		attemptCtx, cancel := context.WithTimeout(ctx, time.Duration(defaultMs(t.config.RequestTimeoutMs, defaultRequestTimeoutMs))*time.Millisecond)
		state, blockTimestamp, err := t.readSnapshot(attemptCtx, common.HexToAddress(p.Address))
		cancel()
		if err == nil {
			extra, marshalErr := json.Marshal(&Extra{State: state, CapturedAtMs: started.UnixMilli(), BlockTimestamp: blockTimestamp,
				BlockTimeMs: defaultMs(t.config.BlockTimeMs, defaultBlockTimeMs), MaxSnapshotAgeMs: defaultMs(t.config.MaxSnapshotAgeMs, defaultMaxSnapshotAgeMs)})
			if marshalErr != nil {
				return p, marshalErr
			}
			tokens := make([]*entity.PoolToken, len(state.Tokens))
			reserves := make(entity.PoolReserves, len(tokens))
			for i, token := range state.Tokens {
				tokens[i] = &entity.PoolToken{Address: token.Address, Decimals: token.Decimals, Swappable: true}
				reserves[i] = state.Balances[i].Dec()
			}
			p.Tokens, p.Reserves, p.Extra = tokens, reserves, string(extra)
			p.BlockNumber, p.Timestamp = state.BlockNumber, started.Unix()
			return p, nil
		}
		last = err
		fields := logger.Fields{"dexType": DexType, "pool": p.Address, "attempt": attempt, "errorKind": classifyRefreshError(err)}
		var detailed *refreshError
		if errors.As(err, &detailed) {
			fields["stage"] = detailed.stage
		}
		logger.WithFields(fields).Warnf("snapshot refresh failed: %v", err)
		if attempt == 1 {
			timer := time.NewTimer(refreshRetryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return p, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return p, last
}

func (t *PoolTracker) readSnapshot(ctx context.Context, proxy common.Address) (State, uint64, error) {
	client := t.ethrpcClient.GetETHClient().Client()
	if !t.config.DisableLens {
		state, timestamp, err := readSnapshotLens(ctx, client, proxy)
		if err == nil {
			return state, timestamp, nil
		}
		if !lensUnsupported(err) {
			return State{}, 0, err
		}
		logger.WithFields(logger.Fields{"dexType": DexType, "pool": hexutil.Encode(proxy[:]), "errorKind": "lens_unsupported"}).Warnf("using number-pinned page reader: %v", err)
	}
	state, err := readSnapshotPage(ctx, client, proxy)
	return state, 0, err
}

func snapshotCall(to common.Address, data []byte) map[string]any {
	// Nonzero simulation gasPrice preserves BASEFEE; this is not a swap fee assumption.
	return map[string]any{"to": to, "data": hexutil.Bytes(data), "gasPrice": "0x1"}
}

// The page selects its own block. Issuers and a confirming page are number-pinned
// to it: no EIP-1898 hash parameter or independent head/header request is needed.
func readSnapshotPage(ctx context.Context, client *rpc.Client, proxy common.Address) (State, error) {
	data, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(maxTokens))
	if err != nil {
		return State{}, err
	}
	var raw hexutil.Bytes
	if err = client.CallContext(ctx, &raw, "eth_call", snapshotCall(proxy, data), "latest"); err != nil {
		return State{}, err
	}
	page, err := decodeSnapshot(raw)
	if err != nil {
		return State{}, refreshFailure("validation", "page decode", err)
	}
	if page.BlockNumber == nil || !page.BlockNumber.IsUint64() || page.TokenCount > maxTokens || len(page.Tokens) != int(page.TokenCount) {
		return State{}, refreshFailure("validation", "page context", ErrInvalidState)
	}
	block := hexutil.EncodeBig(page.BlockNumber)
	var issuers []common.Address
	for _, token := range page.Tokens {
		if isXStock(hexutil.Encode(token.Token[:])) {
			issuers = append(issuers, token.Token)
		}
	}
	sharesData, err := thogAMMABI.Pack("sharesOf", proxy)
	if err != nil {
		return State{}, err
	}
	multiplierData, err := thogAMMABI.Pack("getCurrentMultiplier")
	if err != nil {
		return State{}, err
	}
	results := make([]hexutil.Bytes, 2*len(issuers)+1)
	batch := make([]rpc.BatchElem, len(results))
	for i, issuer := range issuers {
		for j, calldata := range [][]byte{sharesData, multiplierData} {
			index := 2*i + j
			batch[index] = rpc.BatchElem{Method: "eth_call", Args: []any{snapshotCall(issuer, calldata), block}, Result: &results[index]}
		}
	}
	last := len(batch) - 1
	batch[last] = rpc.BatchElem{Method: "eth_call", Args: []any{snapshotCall(proxy, data), block}, Result: &results[last]}
	if err = client.BatchCallContext(ctx, batch); err != nil {
		return State{}, err
	}
	for i, result := range batch {
		if result.Error != nil {
			kind := classifyRefreshError(result.Error)
			if i < last {
				if kind != "provider_lag" {
					kind = "issuer"
				}
				return State{}, refreshFailure(kind, "number-pinned issuer", fmt.Errorf("issuer %s call %d: %w", hexutil.Encode(issuers[i/2][:]), i%2, result.Error))
			}
			return State{}, refreshFailure(kind, "number-pinned batch", result.Error)
		}
	}
	if !bytes.Equal(raw, results[last]) {
		return State{}, refreshFailure("reorg", "page changed at pinned number", ErrInvalidState)
	}
	shares := make(map[string]ShareBalance, len(issuers))
	for i, issuer := range issuers {
		if len(results[2*i]) != 32 || len(results[2*i+1]) != 96 {
			return State{}, refreshFailure("validation", "issuer result", ErrInvalidState)
		}
		var holding ShareBalance
		holding.Shares.SetBytes(results[2*i])
		holding.Multiplier.SetBytes(results[2*i+1][:32])
		shares[hexutil.Encode(issuer[:])] = holding
	}
	state, err := snapshotState(page, shares)
	if err != nil {
		return State{}, refreshFailure("validation", "page snapshot", err)
	}
	return state, nil
}

// Whole-second scheduling is unchanged. MakerPageUpdated can trigger earlier reads.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 { return p.Timestamp + 1 }

var _ pool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)
