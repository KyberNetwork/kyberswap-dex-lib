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

// snapshotRequests holds the per-proxy request payloads; both are immutable.
type snapshotRequests struct {
	lensInit, multicall []byte
}

func (t *PoolTracker) clock() time.Time {
	if t.now != nil {
		return t.now()
	}
	return time.Now()
}

// requests encodes the lens init code and the Multicall3 batch once per proxy.
func (t *PoolTracker) requests(proxy common.Address) (*snapshotRequests, error) {
	if cached, ok := t.built.Load(proxy); ok {
		return cached.(*snapshotRequests), nil
	}
	lensInit, err := lensInitCode(proxy)
	if err != nil {
		return nil, err
	}
	multicall, err := multicallRequest(proxy)
	if err != nil {
		return nil, err
	}
	cached, _ := t.built.LoadOrStore(proxy, &snapshotRequests{lensInit: lensInit, multicall: multicall})
	return cached.(*snapshotRequests), nil
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
	requests, err := t.requests(proxy)
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
	return readSnapshotMulticall(ctx, client, requests.multicall)
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

// readSnapshotMulticall is the fallback for providers that refuse the lens's
// creation call. One Multicall3.tryAggregate at "latest" runs getPoolData, the
// block timestamp and every catalog issuer's sharesOf/getCurrentMultiplier in one
// EVM state, so nothing is pinned by hash or number. Unlisted catalog issuers may
// revert; the page selects which issuer results are required.
var multicall3Address = common.HexToAddress(multicall3)

// multicallCallCount is the batch size: page, timestamp, then two calls per issuer.
const multicallCallCount = 2 + 2*len(xStocks)

// multicallRequest encodes Multicall3.tryAggregate(false, [getPoolData(0, 28),
// getCurrentBlockTimestamp, then sharesOf(proxy) and getCurrentMultiplier per issuer]).
func multicallRequest(proxy common.Address) ([]byte, error) {
	pageData, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(maxTokens))
	if err != nil {
		return nil, err
	}
	sharesData, err := thogAMMABI.Pack("sharesOf", proxy)
	if err != nil {
		return nil, err
	}
	multiplierData, err := thogAMMABI.Pack("getCurrentMultiplier")
	if err != nil {
		return nil, err
	}
	calls := make([]tryCall, 0, multicallCallCount)
	calls = append(calls, tryCall{Target: proxy, CallData: pageData},
		tryCall{Target: multicall3Address, CallData: common.FromHex(selectorGetCurrentBlockTimestamp)})
	for _, token := range xStocks {
		issuer := common.HexToAddress(token)
		calls = append(calls, tryCall{Target: issuer, CallData: sharesData}, tryCall{Target: issuer, CallData: multiplierData})
	}
	return abipkg.Multicall3ABI.Pack(abipkg.Multicall3TryAggregate, false, calls)
}

func readSnapshotMulticall(ctx context.Context, client *rpc.Client, request []byte) (State, uint64, error) {
	var raw hexutil.Bytes
	if err := client.CallContext(ctx, &raw, "eth_call", snapshotCall(multicall3Address, request), "latest"); err != nil {
		return State{}, 0, err
	}
	out, err := abipkg.Multicall3ABI.Unpack(abipkg.Multicall3TryAggregate, raw)
	if err != nil || len(out) != 1 {
		return State{}, 0, refreshFailure("validation", "multicall reply", fmt.Errorf("%w: %v", ErrInvalidState, err))
	}
	results := *abi.ConvertType(out[0], new([]tryResult)).(*[]tryResult)
	if len(results) != multicallCallCount {
		return State{}, 0, refreshFailure("validation", "multicall reply", ErrInvalidState)
	}
	if !results[0].Success {
		return State{}, 0, refreshFailure("contract", "multicall page", fmt.Errorf("getPoolData reverted: 0x%x", results[0].ReturnData))
	}
	page, err := decodeSnapshot(results[0].ReturnData)
	if err != nil {
		return State{}, 0, refreshFailure("validation", "page decode", err)
	}
	// A zero BASEFEE means the node ignored gasPrice; execution friction needs the real one.
	if page.BlockNumber == nil || !page.BlockNumber.IsUint64() || page.BaseFeePerGas == nil || page.BaseFeePerGas.Sign() == 0 ||
		page.TokenCount > maxTokens || len(page.Tokens) != int(page.TokenCount) {
		return State{}, 0, refreshFailure("validation", "page context", ErrInvalidState)
	}
	stamp := results[1]
	if !stamp.Success || len(stamp.ReturnData) != 32 {
		return State{}, 0, refreshFailure("validation", "block timestamp", ErrInvalidState)
	}
	timestamp := new(big.Int).SetBytes(stamp.ReturnData)
	if !timestamp.IsUint64() || timestamp.Sign() <= 0 {
		return State{}, 0, refreshFailure("validation", "block timestamp", ErrInvalidState)
	}
	shares := make(map[string]ShareBalance)
	for _, token := range page.Tokens {
		issuer := hexutil.Encode(token.Token[:])
		i := slices.Index(xStocks[:], issuer)
		if i < 0 {
			continue
		}
		sharesOf, multiplier := results[2+2*i], results[3+2*i]
		if !sharesOf.Success || !multiplier.Success {
			return State{}, 0, refreshFailure("issuer", "multicall issuer", fmt.Errorf("issuer %s: read reverted", issuer))
		}
		if len(sharesOf.ReturnData) != 32 || len(multiplier.ReturnData) != 96 {
			return State{}, 0, refreshFailure("validation", "issuer result", ErrInvalidState)
		}
		var holding ShareBalance
		holding.Shares.SetBytes(sharesOf.ReturnData)
		holding.Multiplier.SetBytes(multiplier.ReturnData[:32])
		shares[issuer] = holding
	}
	state, err := snapshotState(page, shares)
	if err != nil {
		return State{}, 0, refreshFailure("validation", "page snapshot", err)
	}
	return state, timestamp.Uint64(), nil
}

// Whole-second scheduling is unchanged. MakerPageUpdated can trigger earlier reads.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 { return p.Timestamp + 1 }

var _ pool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)
