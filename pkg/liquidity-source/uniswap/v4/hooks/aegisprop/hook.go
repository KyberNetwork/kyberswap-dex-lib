package aegisprop

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const maxLevels = 10

// Level mirrors the on-chain Level struct (integration guide §4 B1). Amplitude is always
// denominated in raw token0 units, on both the Bids and Asks side.
type Level struct {
	Price     *uint256.Int `json:"p"`
	Amplitude *uint256.Int `json:"a"`
}

type LevelRPC struct {
	Price     *big.Int
	Amplitude *big.Int
}

type LadderRPC struct {
	Bids       [maxLevels]LevelRPC
	Asks       [maxLevels]LevelRPC
	Sequence   uint64
	Expiration uint32
	BidLen     uint8
	AskLen     uint8
}

// PoolConfigRPC mirrors the hook's poolConfig(poolId) getter. Oracle is decoded because it's part
// of the same ABI tuple, but only the fee/staleness fields are actually used.
type PoolConfigRPC struct {
	MmId                       uint16
	Token0Decimals             uint8
	Token1Decimals             uint8
	SlackBps                   uint16
	TakerFeeBps                uint16
	HaircutBpsAtStaleThreshold uint16
	FreshThresholdSec          uint32
	StaleThresholdSec          uint32
	Oracle                     OracleConfigRPC
}

type OracleConfigRPC struct {
	FeedA                 common.Address
	FeedADecimals         uint8
	IsCrossRate           bool
	RequiresSequencerFeed bool
	FeedB                 common.Address
	FeedBDecimals         uint8
}

// Hook holds the last-tracked ladder (integration guide §4 Path B) plus the venue's taker fee and
// staleness-haircut config, used to compute indicative swap amounts off-chain during route
// simulation. The ladder walk alone (as documented) omits fees/protective adjustments the on-chain
// hook applies in beforeSwap; TakerFeeBps and the staleness ramp fields replicate that on top.
type Hook struct {
	uniswapv4.Hook `json:"-"`

	Bids []Level `json:"b"`
	Asks []Level `json:"k"`

	Expiration uint64 `json:"e"` // unix ts; ladder valid while now < Expiration

	TakerFeeBps                uint64 `json:"f"`
	HaircutBpsAtStaleThreshold uint64 `json:"h"`
	FreshThresholdSec          uint64 `json:"fs"`
	StaleThresholdSec          uint64 `json:"ss"`
	TrackedAt                  uint64 `json:"t"` // unix ts when Track last ran; used as the staleness-ramp age baseline
}

type SwapInfo struct {
	ZeroForOne bool
	NewLevels  []Level
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4AegisProp},
	}
	_ = param.HookExtra.Unmarshal(hook)
	return hook
}, HookAddresses...)

func (h *Hook) AllowEmptyTicks() bool { return true }

// GetReserves reuses the ladder Track just fetched (Track always runs before GetReserves, see
// pool_tracker.go's resolveHookState) instead of an extra RPC round-trip: reserve0 is the token0
// the venue can sell (sum of Ask amplitudes), reserve1 is the token1 it can spend buying token0
// (sum of Bid amplitudes converted through each level's price).
func (h *Hook) GetReserves(_ context.Context, _ *uniswapv4.HookParam) (entity.PoolReserves, error) {
	reserve0 := new(uint256.Int)
	for _, lvl := range h.Asks {
		if lvl.Amplitude != nil {
			reserve0.Add(reserve0, lvl.Amplitude)
		}
	}

	reserve1 := new(uint256.Int)
	tmp := new(uint256.Int)
	for _, lvl := range h.Bids {
		if lvl.Amplitude == nil || lvl.Price == nil {
			continue
		}
		reserve1.Add(reserve1, u256.MulDivDown(tmp, lvl.Amplitude, lvl.Price, uPriceScale))
	}

	return entity.PoolReserves{reserve0.Dec(), reserve1.Dec()}, nil
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	poolId := common.HexToHash(param.Pool.Address)
	hookAddr := param.HookAddress.Hex()

	var ladder LadderRPC
	var poolConfig PoolConfigRPC
	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    aegisPropHookABI,
		Target: hookAddr,
		Method: "ladder",
		Params: []any{poolId},
	}, []any{&struct{ *LadderRPC }{&ladder}}).AddCall(&ethrpc.Call{
		ABI:    aegisPropHookABI,
		Target: hookAddr,
		Method: "poolConfig",
		Params: []any{poolId},
	}, []any{&poolConfig})

	if _, err := req.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to track aegis propamm ladder: %w", err)
	}

	h.Bids = toLevels(ladder.Bids[:], ladder.BidLen)
	h.Asks = toLevels(ladder.Asks[:], ladder.AskLen)
	h.Expiration = uint64(ladder.Expiration)

	h.TakerFeeBps = uint64(poolConfig.TakerFeeBps)
	h.HaircutBpsAtStaleThreshold = uint64(poolConfig.HaircutBpsAtStaleThreshold)
	h.FreshThresholdSec = uint64(poolConfig.FreshThresholdSec)
	h.StaleThresholdSec = uint64(poolConfig.StaleThresholdSec)
	h.TrackedAt = uint64(time.Now().Unix())

	return json.Marshal(h)
}

func toLevels(rpcLevels []LevelRPC, length uint8) []Level {
	if int(length) > len(rpcLevels) {
		length = uint8(len(rpcLevels))
	}
	levels := make([]Level, 0, length)
	for i := range int(length) {
		price, _ := uint256.FromBig(rpcLevels[i].Price)
		amplitude, _ := uint256.FromBig(rpcLevels[i].Amplitude)
		levels = append(levels, Level{Price: price, Amplitude: amplitude})
	}
	return levels
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	amt := params.AmountSpecified
	if amt == nil || amt.Sign() <= 0 {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
			Gas:              gasBeforeSwap,
		}, nil
	}

	now := uint64(time.Now().Unix())
	if h.Expiration != 0 && now >= h.Expiration {
		return nil, ErrStaleLadder
	}

	age := now - h.TrackedAt
	if now < h.TrackedAt {
		age = 0
	}
	haircutBps, freshEnough := stalenessHaircutBps(age, h.FreshThresholdSec, h.StaleThresholdSec, uint16(h.HaircutBpsAtStaleThreshold))
	if !freshEnough {
		return nil, ErrStaleLadder
	}
	totalBps := h.TakerFeeBps + haircutBps
	if totalBps >= bpsDenom {
		return nil, ErrInvalidAmount
	}

	amtU, overflow := uint256.FromBig(amt)
	if overflow {
		return nil, ErrInvalidAmount
	}

	levels := h.Bids
	if !params.ZeroForOne {
		levels = h.Asks
	}
	if len(levels) == 0 {
		return nil, ErrNoLiquidity
	}

	var (
		deltaSpecified, deltaUnspecified *big.Int
		updated                          []Level
	)
	if params.CalcOut {
		rawOut, upd, filled := walkExactIn(levels, amtU, params.ZeroForOne)
		if !filled {
			return nil, ErrInsufficientLadderDepth
		}
		updated = upd
		out := applyFeeOut(rawOut, totalBps)
		deltaSpecified = new(big.Int).Set(amt)
		deltaUnspecified = new(big.Int).Neg(out.ToBig())
	} else {
		// The taker fee/haircut is applied to the amount the taker pays, so the ladder must be
		// walked for the larger pre-fee input that nets the requested output after the haircut.
		grossAmt := applyFeeIn(amtU, totalBps)
		in, upd, filled := walkExactOut(levels, grossAmt, params.ZeroForOne)
		if !filled {
			return nil, ErrInsufficientLadderDepth
		}
		updated = upd
		deltaSpecified = new(big.Int).Neg(amt)
		deltaUnspecified = new(big.Int).Set(in.ToBig())
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: deltaUnspecified,
		Gas:              gasBeforeSwap,
		SwapInfo:         &SwapInfo{ZeroForOne: params.ZeroForOne, NewLevels: updated},
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	cloned.Bids = cloneLevels(h.Bids)
	cloned.Asks = cloneLevels(h.Asks)
	return &cloned
}

func cloneLevels(levels []Level) []Level {
	if levels == nil {
		return nil
	}
	cloned := make([]Level, len(levels))
	for i, lvl := range levels {
		cloned[i] = Level{Price: lvl.Price.Clone(), Amplitude: lvl.Amplitude.Clone()}
	}
	return cloned
}

func (h *Hook) UpdateBalance(swapInfo any) {
	info, ok := swapInfo.(*SwapInfo)
	if !ok || info == nil {
		return
	}
	if info.ZeroForOne {
		h.Bids = info.NewLevels
	} else {
		h.Asks = info.NewLevels
	}
}

var _ uniswapv4.Hook = (*Hook)(nil)
