package thogprop

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	now          func() time.Time // nil: time.Now

	// lensRefusedUntilMs skips the lens while a provider is known to refuse it,
	// saving one failed round trip (and a warning) per refresh.
	lensRefusedUntilMs atomic.Int64
	built              sync.Map // common.Address → *snapshotRequests
}

// snapshotRequests holds one proxy's request payloads. The lens needs only the
// proxy; the Multicall3 batch also names every listed token, so it is rebuilt
// when the token list changes. A stored value is immutable.
type snapshotRequests struct {
	lensInit  []byte
	tokens    []common.Address
	multicall []byte
}

func (t *PoolTracker) clock() time.Time {
	if t.now != nil {
		return t.now()
	}
	return time.Now()
}

// requests returns the payloads for reading proxy with the given token list,
// encoding them only when the proxy is new or its token list has changed.
func (t *PoolTracker) requests(proxy common.Address, tokens []common.Address) (*snapshotRequests, error) {
	cached, _ := t.built.Load(proxy)
	previous, _ := cached.(*snapshotRequests)
	if previous != nil && slices.Equal(previous.tokens, tokens) {
		return previous, nil
	}
	next := &snapshotRequests{tokens: slices.Clone(tokens)}
	var err error
	if previous != nil {
		next.lensInit = previous.lensInit
	} else if next.lensInit, err = lensInitCode(proxy); err != nil {
		return nil, err
	}
	if next.multicall, err = multicallRequest(proxy, tokens); err != nil {
		return nil, err
	}
	t.built.Store(proxy, next)
	return next, nil
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

// A failed read is retried whole; a snapshot never combines two reads. Each
// attempt has one deadline, including a switch of reader; at most 4.3 s by default.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	// The token list from discovery or the previous refresh; the Multicall3 reader
	// asks the pool about each of them.
	tokens := make([]common.Address, len(p.Tokens))
	for i, token := range p.Tokens {
		tokens[i] = common.HexToAddress(token.Address)
	}
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return p, err
		}
		started := time.Now()
		attemptCtx, cancel := context.WithTimeout(ctx, time.Duration(defaultMs(t.config.RequestTimeoutMs, defaultRequestTimeoutMs))*time.Millisecond)
		state, blockTimestamp, err := t.readSnapshot(attemptCtx, common.HexToAddress(p.Address), tokens)
		cancel()
		if err == nil {
			extra, marshalErr := json.Marshal(&Extra{State: state, CapturedAtMs: started.UnixMilli(), BlockTimestamp: blockTimestamp,
				BlockTimeMs: defaultMs(t.config.BlockTimeMs, defaultBlockTimeMs), MaxSnapshotAgeMs: defaultMs(t.config.MaxSnapshotAgeMs, defaultMaxSnapshotAgeMs)})
			if marshalErr != nil {
				return p, marshalErr
			}
			poolTokens := make([]*entity.PoolToken, len(state.Tokens))
			reserves := make(entity.PoolReserves, len(poolTokens))
			for i, token := range state.Tokens {
				poolTokens[i] = &entity.PoolToken{Address: token.Address, Decimals: token.Decimals, Swappable: true}
				reserves[i] = state.Balances[i].Dec()
			}
			p.Tokens, p.Reserves, p.Extra = poolTokens, reserves, string(extra)
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

func (t *PoolTracker) readSnapshot(ctx context.Context, proxy common.Address, tokens []common.Address) (State, uint64, error) {
	client := t.ethrpcClient.GetETHClient().Client()
	requests, err := t.requests(proxy, tokens)
	if err != nil {
		return State{}, 0, err
	}
	if !t.config.DisableLens && t.clock().UnixMilli() >= t.lensRefusedUntilMs.Load() {
		state, timestamp, err := readSnapshotLens(ctx, client, proxy, requests.lensInit)
		if err == nil {
			return state, timestamp, nil
		}
		if !lensUnsupported(err) {
			return State{}, 0, err
		}
		// Capability refusals persist per provider: probe the lens again only after
		// the cooldown, which also bounds the warning to one per cooldown.
		t.lensRefusedUntilMs.Store(t.clock().Add(lensRetryAfter).UnixMilli())
		logger.WithFields(logger.Fields{"dexType": DexType, "pool": hexutil.Encode(proxy[:]), "errorKind": "lens_unsupported"}).
			Warnf("using Multicall3 reader for %s: %v", lensRetryAfter, err)
	}
	state, timestamp, listed, err := readSnapshotMulticall(ctx, client, proxy, tokens, requests.multicall)
	if err != nil || slices.Equal(listed, tokens) {
		return state, timestamp, err
	}
	// The pool lists different tokens than the batch asked about (first read, or a
	// listing changed). Read once more with the pool's own list; a list that moves
	// again fails this attempt rather than combining two reads.
	if requests, err = t.requests(proxy, listed); err != nil {
		return State{}, 0, err
	}
	state, timestamp, relisted, err := readSnapshotMulticall(ctx, client, proxy, listed, requests.multicall)
	if err == nil && !slices.Equal(relisted, listed) {
		err = refreshFailure("validation", "multicall token list", ErrInvalidState)
	}
	return state, timestamp, err
}

func snapshotCall(to common.Address, data []byte) map[string]any {
	// Nonzero simulation gasPrice preserves BASEFEE; this is not a swap fee assumption.
	return map[string]any{"to": to, "data": hexutil.Bytes(data), "gasPrice": "0x1"}
}

type tryCall struct {
	Target   common.Address
	CallData []byte
}

type tryResult struct {
	Success    bool
	ReturnData []byte
}

var multicall3Address = common.HexToAddress(multicall3)

// The batch starts with the page and the block timestamp, then holds two calls
// per token.
const multicallFixedCalls = 2

// multicallRequest encodes Multicall3.tryAggregate(false, [getPoolData(0, 28),
// getCurrentBlockTimestamp(), then per token the pool's storedMultiplier(token)
// and the token's sharesOf(pool)]). Only share-based tokens implement sharesOf;
// its result is read only where the multiplier is nonzero.
func multicallRequest(proxy common.Address, tokens []common.Address) ([]byte, error) {
	pageData, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(maxTokens))
	if err != nil {
		return nil, err
	}
	sharesData, err := thogAMMABI.Pack("sharesOf", proxy)
	if err != nil {
		return nil, err
	}
	calls := make([]tryCall, 0, multicallFixedCalls+2*len(tokens))
	calls = append(calls, tryCall{Target: proxy, CallData: pageData},
		tryCall{Target: multicall3Address, CallData: common.FromHex(selectorGetCurrentBlockTimestamp)})
	for _, token := range tokens {
		multiplierData, err := thogAMMABI.Pack("storedMultiplier", token)
		if err != nil {
			return nil, err
		}
		calls = append(calls, tryCall{Target: proxy, CallData: multiplierData}, tryCall{Target: token, CallData: sharesData})
	}
	return abipkg.Multicall3ABI.Pack(abipkg.Multicall3TryAggregate, false, calls)
}

// readSnapshotMulticall is the reader for providers that refuse the lens's
// creation call. One Multicall3.tryAggregate at "latest" runs every read in one
// EVM state, so nothing is pinned by hash or number. It returns the pool's token
// list; the state is valid only when that list equals the tokens the batch asked
// about, because share-based holdings are read per requested token.
func readSnapshotMulticall(ctx context.Context, client *rpc.Client, proxy common.Address, tokens []common.Address, request []byte) (State, uint64, []common.Address, error) {
	var raw hexutil.Bytes
	if err := client.CallContext(ctx, &raw, "eth_call", snapshotCall(multicall3Address, request), "latest"); err != nil {
		return State{}, 0, nil, err
	}
	out, err := abipkg.Multicall3ABI.Unpack(abipkg.Multicall3TryAggregate, raw)
	if err != nil || len(out) != 1 {
		return State{}, 0, nil, refreshFailure("validation", "multicall reply", fmt.Errorf("%w: %v", ErrInvalidState, err))
	}
	results := *abi.ConvertType(out[0], new([]tryResult)).(*[]tryResult)
	if len(results) != multicallFixedCalls+2*len(tokens) {
		return State{}, 0, nil, refreshFailure("validation", "multicall reply", ErrInvalidState)
	}
	if !results[0].Success {
		return State{}, 0, nil, refreshFailure("contract", "multicall page", fmt.Errorf("getPoolData reverted: 0x%x", results[0].ReturnData))
	}
	page, err := decodeSnapshot(results[0].ReturnData)
	if err != nil {
		return State{}, 0, nil, refreshFailure("validation", "page decode", err)
	}
	// A zero BASEFEE means the node ignored gasPrice; execution friction needs the real one.
	if page.BlockNumber == nil || !page.BlockNumber.IsUint64() || page.BaseFeePerGas == nil || page.BaseFeePerGas.Sign() == 0 ||
		page.TokenCount > maxTokens || len(page.Tokens) != int(page.TokenCount) {
		return State{}, 0, nil, refreshFailure("validation", "page context", ErrInvalidState)
	}
	listed := make([]common.Address, len(page.Tokens))
	for i, token := range page.Tokens {
		listed[i] = token.Token
	}
	if !slices.Equal(listed, tokens) {
		return State{}, 0, listed, nil
	}
	stamp := results[1]
	if !stamp.Success || len(stamp.ReturnData) != 32 {
		return State{}, 0, nil, refreshFailure("validation", "block timestamp", ErrInvalidState)
	}
	timestamp := new(big.Int).SetBytes(stamp.ReturnData)
	if !timestamp.IsUint64() || timestamp.Sign() <= 0 {
		return State{}, 0, nil, refreshFailure("validation", "block timestamp", ErrInvalidState)
	}
	shares := make(map[string]ShareBalance)
	for i, token := range tokens {
		multiplier, sharesOf := results[multicallFixedCalls+2*i], results[multicallFixedCalls+2*i+1]
		if !multiplier.Success {
			return State{}, 0, nil, refreshFailure("contract", "multicall stored multiplier", fmt.Errorf("token %s: read reverted", token))
		}
		if len(multiplier.ReturnData) != 32 {
			return State{}, 0, nil, refreshFailure("validation", "stored multiplier result", ErrInvalidState)
		}
		var holding ShareBalance
		if holding.Multiplier.SetBytes(multiplier.ReturnData); holding.Multiplier.IsZero() {
			continue // An ordinary ERC-20: the page balance is its balanceOf.
		}
		if !sharesOf.Success {
			return State{}, 0, nil, refreshFailure("issuer", "multicall shares", fmt.Errorf("token %s: sharesOf reverted", token))
		}
		if len(sharesOf.ReturnData) != 32 {
			return State{}, 0, nil, refreshFailure("validation", "shares result", ErrInvalidState)
		}
		holding.Shares.SetBytes(sharesOf.ReturnData)
		shares[hexutil.Encode(token[:])] = holding
	}
	state, err := snapshotState(page, shares)
	if err != nil {
		return State{}, 0, nil, refreshFailure("validation", "page snapshot", err)
	}
	return state, timestamp.Uint64(), listed, nil
}

// Refresh again after one second; a MakerPageUpdated event can prompt an earlier read.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 { return p.Timestamp + 1 }

var _ pool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)
