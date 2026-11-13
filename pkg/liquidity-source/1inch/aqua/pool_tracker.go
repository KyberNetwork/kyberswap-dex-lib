package oneinchaqua

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	_ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

	_ pool.IPoolTracker           = (*PoolTracker)(nil)
	_ pool.IBatchPoolStateTracker = (*PoolTracker)(nil)
)

// PoolTracker refreshes strategies. It keeps no state between calls: a pool may be refreshed by any
// tracker instance, so everything a refresh needs is read in the refresh.
type PoolTracker struct {
	aqua          common.Address
	confirmations uint64
	ethrpcClient  *ethrpc.Client
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	s, err := cfg.trackerSettings()
	if err != nil {
		return nil, err
	}
	return &PoolTracker{aqua: s.aqua, confirmations: s.confirmations, ethrpcClient: ethrpcClient}, nil
}

// rawBalancesOutput is the (balance, tokensCount) pair of Aqua.rawBalances.
type rawBalancesOutput struct {
	Balance     *big.Int
	TokensCount *big.Int
}

// trackedPool is one pool of a batch with the positions of its reads in the multicall.
type trackedPool struct {
	index  int
	pool   entity.Pool
	old    Extra
	maker  common.Address
	app    common.Address
	hash   [32]byte
	tokens [2]common.Address
	raw    [2]rawBalancesOutput
	rawAt  [2]int
	wallet [2]*walletRead
}

// walletRead is the maker's balanceOf and allowance to Aqua for one token, shared by every pool of the
// maker in the batch that holds the token.
type walletRead struct {
	balance, allowance *big.Int
	balanceAt          int
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	pools, errs := t.GetNewPoolStates(ctx, []entity.Pool{p})
	return pools[0], errs[0]
}

// GetNewPoolStates reads the batch in one multicall at one block: rawBalances per pool, and balanceOf and
// allowance per (maker, token). A transport error or a failed rawBalances read keeps the stored state; a
// failed wallet read gives that token a zero limit and reserve. tokensCount other than 2 is Inactive;
// 0xff is Docked, final once read docked `confirmations` blocks later (then Timestamp 1, no more RPC).
func (t *PoolTracker) GetNewPoolStates(ctx context.Context, pools []entity.Pool) ([]entity.Pool, []error) {
	out := make([]entity.Pool, len(pools))
	errs := make([]error, len(pools))

	var tracked []*trackedPool
	for i, p := range pools {
		out[i] = p
		tp, docked, err := newTrackedPool(i, p)
		switch {
		case err != nil:
			errs[i] = err
		case docked: // permanent: the stored pool stands, no RPC
		default:
			tracked = append(tracked, tp)
		}
	}
	if len(tracked) == 0 {
		return out, errs
	}

	wallets := map[[2]common.Address]*walletRead{}
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	aquaAddr := hexutil.Encode(t.aqua[:])
	calls := 0
	for _, tp := range tracked {
		for k, token := range tp.tokens {
			tp.rawAt[k] = calls
			calls++
			req.AddCall(&ethrpc.Call{ABI: aquaABI, Target: aquaAddr, Method: rawBalancesMethod.Name,
				Params: []any{tp.maker, tp.app, tp.hash, token}}, []any{&tp.raw[k]})
			key := [2]common.Address{tp.maker, token}
			w, ok := wallets[key]
			if !ok {
				w = &walletRead{balanceAt: calls}
				wallets[key] = w
				calls += 2
				tokenAddr := hexutil.Encode(token[:])
				req.AddCall(&ethrpc.Call{ABI: erc20ABI, Target: tokenAddr, Method: balanceOfMethod.Name,
					Params: []any{tp.maker}}, []any{&w.balance})
				req.AddCall(&ethrpc.Call{ABI: erc20ABI, Target: tokenAddr, Method: allowanceMethod.Name,
					Params: []any{tp.maker, t.aqua}}, []any{&w.allowance})
			}
			tp.wallet[k] = w
		}
	}

	resp, err := req.TryBlockAndAggregate()
	if err == nil && (resp == nil || resp.BlockNumber == nil || resp.BlockNumber.Sign() <= 0 || len(resp.Result) != calls) {
		err = errors.New("1inch-aqua: malformed multicall response")
	}
	if err != nil {
		for _, tp := range tracked {
			errs[tp.index] = err
		}
		return out, errs
	}
	block := resp.BlockNumber.Uint64()
	now := time.Now().Unix()
	for _, tp := range tracked {
		out[tp.index], errs[tp.index] = tp.build(resp.Result, block, now, t.confirmations)
	}
	return out, errs
}

// newTrackedPool parses a stored pool. docked reports a pool whose dock is final: it is not read again.
func newTrackedPool(index int, p entity.Pool) (*trackedPool, bool, error) {
	if len(p.Tokens) != 2 {
		return nil, false, ErrInvalidState
	}
	var static StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &static); err != nil {
		return nil, false, err
	}
	var old Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &old); err != nil {
			return nil, false, err
		}
	}
	if old.Docked && old.DockedFinal {
		return nil, true, nil
	}
	if !common.IsHexAddress(static.Maker) || !common.IsHexAddress(static.App) || !common.IsHexAddress(p.Tokens[0].Address) ||
		!common.IsHexAddress(p.Tokens[1].Address) {
		return nil, false, ErrInvalidState
	}
	hash, err := hexutil.Decode(static.StrategyHash)
	if err != nil || len(hash) != 32 {
		return nil, false, ErrInvalidState
	}
	tp := &trackedPool{index: index, pool: p, old: old, maker: common.HexToAddress(static.Maker), app: common.HexToAddress(static.App)}
	copy(tp.hash[:], hash)
	for k := range 2 {
		tp.tokens[k] = common.HexToAddress(p.Tokens[k].Address)
	}
	return tp, false, nil
}

// build turns the results of one batch into the new pool.
func (tp *trackedPool) build(ok []bool, block uint64, now int64, confirmations uint64) (entity.Pool, error) {
	p := tp.pool
	p.BlockNumber = block

	// tokensCount comes with the raw balances: 0xff on either token is a docked strategy (permanent once final),
	// anything but 2 a strategy that is not (or no longer) shipped.
	rawOK := true
	var docked, inactive bool
	for k := range 2 {
		r := tp.raw[k]
		if !ok[tp.rawAt[k]] || r.Balance == nil || r.TokensCount == nil || r.Balance.Sign() < 0 || r.Balance.BitLen() > 248 {
			rawOK = false
			continue
		}
		switch {
		case !r.TokensCount.IsUint64():
			inactive = true
		case r.TokensCount.Uint64() == tokensCountDocked:
			docked = true
		case r.TokensCount.Uint64() != 2:
			inactive = true
		}
	}

	if !rawOK {
		// The state of the strategy is unknown: keep the stored pool and report it, as for a failed batch.
		return tp.pool, ErrRawRead
	}
	zero := func() *uint256.Int { return new(uint256.Int) }
	extra := Extra{RawBlock: block, WalletBlock: block,
		RawBalance0: zero(), RawBalance1: zero(), WalletBalance0: zero(), WalletBalance1: zero(), Allowance0: zero(), Allowance1: zero()}
	switch {
	case docked:
		extra.Docked = true
		extra.DockedAtBlock = block
		if tp.old.Docked && tp.old.DockedAtBlock > 0 && tp.old.DockedAtBlock < block {
			extra.DockedAtBlock = tp.old.DockedAtBlock
		}
		extra.DockedFinal = block-extra.DockedAtBlock >= confirmations
		// A dock that is not final may be reorged out: keep it active until it is final (Timestamp 1).
		ts := now
		if extra.DockedFinal {
			ts = 1
		}
		return tp.finish(p, &extra, entity.PoolReserves{"0", "0"}, ts)
	case inactive:
		extra.Inactive = true
		// Timestamp is the first inactive read, so the indexer archives it after the inactivity duration.
		ts := now
		if tp.old.Inactive && tp.pool.Timestamp > 1 {
			ts = tp.pool.Timestamp
		}
		return tp.finish(p, &extra, entity.PoolReserves{"0", "0"}, ts)
	}

	raws := [2]*uint256.Int{uint256.MustFromBig(tp.raw[0].Balance), uint256.MustFromBig(tp.raw[1].Balance)}
	extra.RawBalance0, extra.RawBalance1 = raws[0], raws[1]
	balances := [2]*uint256.Int{extra.WalletBalance0, extra.WalletBalance1}
	allowances := [2]*uint256.Int{extra.Allowance0, extra.Allowance1}
	reserves := entity.PoolReserves{"0", "0"}
	for k := range 2 {
		bal, allow, good := walletLimit(tp.wallet[k], ok)
		if !good {
			continue // fail closed for this token: zero limit, zero reserve
		}
		balances[k].Set(bal)
		allowances[k].Set(allow)
		reserves[k] = big256.Min(raws[k], big256.Min(bal, allow)).Dec()
	}
	return tp.finish(p, &extra, reserves, now)
}

// walletLimit decodes one token's wallet read. good is false when either call failed or returned a value
// that is not a uint256.
func walletLimit(w *walletRead, ok []bool) (balance, allowance *uint256.Int, good bool) {
	if !ok[w.balanceAt] || !ok[w.balanceAt+1] || w.balance == nil || w.allowance == nil {
		return nil, nil, false
	}
	b, overflowB := uint256.FromBig(w.balance)
	a, overflowA := uint256.FromBig(w.allowance)
	if overflowB || overflowA || w.balance.Sign() < 0 || w.allowance.Sign() < 0 {
		return nil, nil, false
	}
	return b, a, true
}

func (tp *trackedPool) finish(p entity.Pool, extra *Extra, reserves entity.PoolReserves, timestamp int64) (entity.Pool, error) {
	tp.keepReadBlocks(extra)
	b, err := json.Marshal(extra)
	if err != nil {
		return tp.pool, err
	}
	p.Extra = string(b)
	p.Reserves = reserves
	p.Timestamp = timestamp
	return p, nil
}

// keepReadBlocks keeps the stored RawBlock and WalletBlock when nothing else in Extra changed, so an unchanged
// strategy gives the same Extra and a store that skips unchanged pools does not rewrite it. The blocks are then the
// first read of the current state; entity.Pool.BlockNumber is still the block of this read.
func (tp *trackedPool) keepReadBlocks(extra *Extra) {
	if tp.old.WalletBlock == 0 || tp.old.RawBlock != tp.old.WalletBlock {
		return
	}
	kept := *extra
	kept.RawBlock, kept.WalletBlock = tp.old.RawBlock, tp.old.WalletBlock
	if b, err := json.Marshal(&kept); err == nil && string(b) == tp.pool.Extra {
		*extra = kept
	}
}
