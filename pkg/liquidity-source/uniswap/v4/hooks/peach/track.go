package peach

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

type pokeRPC struct {
	Fee0For1 *big.Int `abi:"fee0For1"`
	Fee1For0 *big.Int `abi:"fee1For0"`
	Expiry   *big.Int `abi:"expiry"`
}

type midRPC struct {
	Enabled   bool     `abi:"enabled"`
	Threshold *big.Int `abi:"midThresholdPips"`
	Fraction  *big.Int `abi:"captureFractionE6"`
	Cap       *big.Int `abi:"midCap"`
}

func bounded(value *big.Int, max uint64) bool {
	return value != nil && value.Sign() >= 0 && value.IsUint64() && value.Uint64() <= max
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	d, found := deployments[param.HookAddress]
	if !found || param.Cfg == nil || param.Cfg.ChainID != d.ChainID {
		return nil, ErrDeployment
	}
	if param.RpcClient == nil || param.Pool == nil {
		return nil, ErrUntracked
	}
	poolID := common.HexToHash(param.Pool.Address)
	target := hexutil.Encode(param.HookAddress[:])
	var fee0, fee1, auto0, auto1, floor, cap, absolute, paused, poolPaused, timestamp *big.Int
	var manager common.Address
	var poke pokeRPC
	var mid midRPC
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).SetOverrides(param.Overrides)
	for _, call := range []struct {
		method string
		args   []any
		result any
	}{
		{"currentFee", []any{poolID, true}, &fee0}, {"currentFee", []any{poolID, false}, &fee1},
		{"autonomousFee", []any{poolID, true}, &auto0}, {"autonomousFee", []any{poolID, false}, &auto1},
		{"pokeFloor", []any{poolID}, &floor}, {"maxFee", []any{poolID}, &cap},
		{"ABSOLUTE_MAX_FEE", nil, &absolute}, {"pokeOf", []any{poolID}, &poke},
		{"pausedUntil", nil, &paused}, {"pausedUntilOf", []any{poolID}, &poolPaused},
		{"midBlockConfig", []any{poolID}, &mid}, {"poolManager", nil, &manager},
	} {
		req.AddCall(&ethrpc.Call{ABI: hookABI, Target: target, Method: call.method, Params: call.args}, []any{call.result})
	}
	clock := param.Cfg.Multicall3Address
	if clock == "" {
		clock = "0xcA11bde05977b3631167028862bE2a173976CA11"
	}
	req.AddCall(&ethrpc.Call{ABI: clockABI, Target: clock, Method: "getCurrentBlockTimestamp"}, []any{&timestamp})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	// Actual deployment observations: BSC and Arc each use their canonical Uniswap v4 manager.
	expectedManager, ok := uniswapv4.PoolManager[param.Cfg.ChainID]
	if !ok || manager != expectedManager {
		return nil, ErrDeployment
	}
	for _, value := range []*big.Int{fee0, fee1, floor, cap, absolute, poke.Fee0For1, poke.Fee1For0, mid.Fraction, mid.Cap} {
		if !bounded(value, feePrecision) {
			return nil, ErrInvalidState
		}
	}
	for _, value := range []*big.Int{auto0, auto1} {
		// RWA opening peaks can exceed the final cap (e.g. closedSpike=3e6).
		// Contract peaks plus directional premium fit uint32; cap only in resolveFee.
		if !bounded(value, 1<<32-1) {
			return nil, ErrInvalidState
		}
	}
	for _, value := range []*big.Int{poke.Expiry, paused, poolPaused, timestamp} {
		if !bounded(value, 1<<63-1) {
			return nil, ErrInvalidState
		}
	}
	if !bounded(mid.Threshold, feePrecision) || floor.Uint64() < 100 || floor.Cmp(cap) > 0 || cap.Cmp(absolute) > 0 ||
		absolute.Sign() == 0 || timestamp.Sign() == 0 || (mid.Enabled && mid.Threshold.Sign() == 0) || mid.Cap.Cmp(absolute) > 0 {
		return nil, ErrInvalidState
	}
	extra := Extra{
		Tracked: true, Timestamp: timestamp.Uint64(), Fee0For1: uint32(fee0.Uint64()), Fee1For0: uint32(fee1.Uint64()),
		Autonomous0For1: uint32(auto0.Uint64()), Autonomous1For0: uint32(auto1.Uint64()),
		Floor: uint32(floor.Uint64()), Cap: uint32(cap.Uint64()), AbsoluteCap: uint32(absolute.Uint64()),
		Poke0For1: uint32(poke.Fee0For1.Uint64()), Poke1For0: uint32(poke.Fee1For0.Uint64()), PokeExpiry: poke.Expiry.Uint64(),
		PausedUntil: paused.Uint64(), PoolPausedUntil: poolPaused.Uint64(),
		Mid: MidBlockConfig{mid.Enabled, uint32(mid.Threshold.Uint64()), uint32(mid.Fraction.Uint64()), uint32(mid.Cap.Uint64())},
	}
	// Check that the sampled getters and our ordinary-fee port agree in this snapshot.
	active := extra.Timestamp < extra.PokeExpiry
	if resolveFee(extra.Autonomous0For1, extra.Floor, extra.Cap, extra.Poke0For1, active) != extra.Fee0For1 ||
		resolveFee(extra.Autonomous1For0, extra.Floor, extra.Cap, extra.Poke1For0, active) != extra.Fee1For0 {
		return nil, ErrInvalidState
	}
	h.Extra, h.RWA, h.ValidDeployment = extra, d.RWA, true
	h.Advanced, h.OpenPrice = false, [4]uint64{}
	return json.Marshal(h)
}
