package metronomeswap

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	defer func(startTime time.Time) {
		logger.WithFields(logger.Fields{
			"exchange": p.Exchange,
			"address":  p.Address,
			"duration": time.Since(startTime).Milliseconds(),
		}).Info("finished GetNewPoolState")
	}(time.Now())

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	// Round 1: pool-wide flags + the current fee-provider/master-oracle addresses (rarely
	// rotated, but re-read every cycle since the package is small and this is cheap).
	var isSwapActive, poolPaused, poolStopped, registryPaused, registryStopped bool
	var feeProvider, masterOracle common.Address

	req1 := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI: poolABI, Target: p.Address, Method: poolMethodIsSwapActive,
	}, []any{&isSwapActive}).AddCall(&ethrpc.Call{
		ABI: poolABI, Target: p.Address, Method: poolMethodPaused,
	}, []any{&poolPaused}).AddCall(&ethrpc.Call{
		ABI: poolABI, Target: p.Address, Method: poolMethodEverythingStopped,
	}, []any{&poolStopped}).AddCall(&ethrpc.Call{
		ABI: poolABI, Target: p.Address, Method: poolMethodFeeProvider,
	}, []any{&feeProvider}).AddCall(&ethrpc.Call{
		ABI: poolABI, Target: p.Address, Method: poolMethodMasterOracle,
	}, []any{&masterOracle}).AddCall(&ethrpc.Call{
		ABI: poolRegistryABI, Target: staticExtra.PoolRegistry, Method: poolRegistryMethodPaused,
	}, []any{&registryPaused}).AddCall(&ethrpc.Call{
		ABI: poolRegistryABI, Target: staticExtra.PoolRegistry, Method: poolRegistryMethodEverythingStopped,
	}, []any{&registryStopped})

	resp1, err := req1.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{"exchange": p.Exchange, "error": err}).
			Error("metronome-swap: round-1 rpc call error")
		return entity.Pool{}, err
	}

	nTokens := len(p.Tokens)
	feeProviderStr := hexutil.Encode(feeProvider[:])
	masterOracleStr := hexutil.Encode(masterOracle[:])

	// Round 2: per-token active/cap/price + every ordered-pair fee, all against the addresses
	// round 1 just confirmed are current. Uses TryAggregate (not Aggregate) because a single
	// synthetic can have no configured price feed (observed live: MasterOracle reverts
	// "Feed not found" for an isActive()==true token) — one bad call must not sink the whole
	// pool's state refresh.
	isActive := make([]bool, nTokens)
	maxTotalSupply := make([]*big.Int, nTokens)
	totalSupply := make([]*big.Int, nTokens)
	priceInUsd := make([]*big.Int, nTokens)

	const callsPerToken = 4
	req2 := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	for i, token := range p.Tokens {
		req2.AddCall(&ethrpc.Call{
			ABI: syntheticTokenABI, Target: token.Address, Method: syntheticTokenMethodIsActive,
		}, []any{&isActive[i]}).AddCall(&ethrpc.Call{
			ABI: syntheticTokenABI, Target: token.Address, Method: syntheticTokenMethodMaxTotalSupply,
		}, []any{&maxTotalSupply[i]}).AddCall(&ethrpc.Call{
			ABI: syntheticTokenABI, Target: token.Address, Method: syntheticTokenMethodTotalSupply,
		}, []any{&totalSupply[i]}).AddCall(&ethrpc.Call{
			ABI: masterOracleABI, Target: masterOracleStr, Method: masterOracleMethodGetPriceInUsd,
			Params: []any{common.HexToAddress(token.Address)},
		}, []any{&priceInUsd[i]})
	}

	// FeeProvider.swapFees is keyed by ordered pair and NOT assumed symmetric — every
	// directed pair among this pool's tokens gets its own read.
	type pairKey struct{ i, j int }
	pairs := make([]pairKey, 0, nTokens*(nTokens-1))
	swapFeesBps := make([]*big.Int, 0, nTokens*(nTokens-1))
	for i, tokenIn := range p.Tokens {
		for j, tokenOut := range p.Tokens {
			if i == j {
				continue
			}
			pairs = append(pairs, pairKey{i, j})
			var fee *big.Int
			swapFeesBps = append(swapFeesBps, fee)
			req2.AddCall(&ethrpc.Call{
				ABI: feeProviderABI, Target: feeProviderStr, Method: feeProviderMethodSwapFees,
				Params: []any{common.HexToAddress(tokenIn.Address), common.HexToAddress(tokenOut.Address)},
			}, []any{&swapFeesBps[len(swapFeesBps)-1]})
		}
	}

	resp2, err := req2.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{"exchange": p.Exchange, "error": err}).
			Error("metronome-swap: round-2 rpc call error")
		return entity.Pool{}, err
	}

	extra := Extra{
		SwapActive:   isSwapActive && !poolPaused && !poolStopped && !registryPaused && !registryStopped,
		FeeProvider:  strings.ToLower(feeProviderStr),
		MasterOracle: strings.ToLower(masterOracleStr),
		Tokens:       make(map[string]TokenState, nTokens),
		SwapFeesBps:  make(map[string]*uint256.Int, len(pairs)),
	}

	reserves := make(entity.PoolReserves, nTokens)
	for i, token := range p.Tokens {
		base := i * callsPerToken
		tokenOk := resp2.Result[base] && resp2.Result[base+1] && resp2.Result[base+2] && resp2.Result[base+3]

		// A failed call (per resp2.Result) leaves its destination *big.Int nil — guard before
		// converting, since uint256.FromBig on a nil *big.Int is undefined behavior.
		maxSupply := safeFromBig(maxTotalSupply[i])
		supply := safeFromBig(totalSupply[i])
		price := safeFromBig(priceInUsd[i])

		if !tokenOk {
			logger.WithFields(logger.Fields{"exchange": p.Exchange, "token": token.Address}).
				Warn("metronome-swap: a synthetic-token read failed (e.g. no oracle feed) — marking token inactive for this refresh")
		}

		extra.Tokens[token.Address] = TokenState{
			// A failed read (most commonly: no configured oracle feed) means this token can't
			// be safely priced or capacity-checked this cycle — treat as inactive rather than
			// quoting off a zero-value price/cap.
			IsActive:       tokenOk && isActive[i],
			MaxTotalSupply: maxSupply,
			TotalSupply:    supply,
			PriceInUsd:     price,
		}
		reserves[i] = headroom(maxSupply, supply).ToBig().String()
	}

	// pairCallsBase begins the pair-fee section of resp2.Result.
	pairCallsBase := nTokens * callsPerToken
	for k, pair := range pairs {
		if !resp2.Result[pairCallsBase+k] {
			continue // leave unset -> CalcAmountOut treats a missing pair as zero fee, matching FeeProvider's own mapping default
		}
		tokenIn, tokenOut := p.Tokens[pair.i].Address, p.Tokens[pair.j].Address
		extra.SwapFeesBps[tokenIn+"-"+tokenOut] = big256.FromBig(swapFeesBps[k])
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Timestamp = time.Now().Unix()
	if resp2.BlockNumber != nil {
		p.BlockNumber = resp2.BlockNumber.Uint64()
	} else if resp1.BlockNumber != nil {
		p.BlockNumber = resp1.BlockNumber.Uint64()
	}

	return p, nil
}

// safeFromBig converts a *big.Int that may be nil (a TryAggregate call that failed leaves its
// destination unset) into a zero uint256.Int instead of passing nil through to big256.FromBig.
func safeFromBig(b *big.Int) *uint256.Int {
	if b == nil {
		return uint256.NewInt(0)
	}
	return big256.FromBig(b)
}
