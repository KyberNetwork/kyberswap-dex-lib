package premium

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrExactOutputDisabled = errors.New("premium: PremiumLaunchHook disables exact-output swaps (ExactOutputDisabled)")
	ErrPoolPaused          = errors.New("premium: pool is paused (PremiumLaunchHook.poolConfig.paused)")
)

// Hook prices Premium's graduated meme/stock pools on Robinhood Chain.
//
// PremiumLaunchHook charges a flat 1% fee on the "desk" (non-meme) side of every trade
// (mirrors PremiumLaunchHook.sol's _beforeSwap/_afterSwap):
//
//   - Exact-output is disabled on-chain (ExactOutputDisabled) - CalcOut=false is an error here.
//   - When the desk currency is the swap's SPECIFIED (input) side - i.e. a "buy" leg paying
//     desk-in - the fee is taken up front in beforeSwap: floor(specifiedIn * 1%), reducing
//     the amount that actually reaches the underlying curve. No afterSwap fee on this leg.
//   - When the desk currency is the UNSPECIFIED (output) side - i.e. a "sell" leg receiving
//     desk-out - beforeSwap does nothing and the fee is taken in afterSwap instead:
//     floor(desk_output * 1%), reducing what the trader actually receives.
//
// MemeIsCurrency0/Paused are read once per Track from the hook's own poolConfig(poolId)
// mapping (immutable + dynamic respectively) - see constant.go for why this hook, unlike
// Fables, is a single shared contract rather than one-per-pool.
type Hook struct {
	uniswapv4.Hook `json:"-"`
	// MemeIsCurrency0 is immutable once a pool is registered (set at graduation).
	MemeIsCurrency0 bool `json:"m0"`
	// Paused mirrors PremiumLaunchHook.poolConfig(poolId).paused - dynamic, polled every Track.
	Paused bool `json:"pa"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Prm},
	}
	var extra Hook
	if err := param.HookExtra.Unmarshal(&extra); err == nil {
		hook.MemeIsCurrency0 = extra.MemeIsCurrency0
		hook.Paused = extra.Paused
	}
	return hook
}, HookAddresses...)

// poolConfigResult mirrors PremiumLaunchHook.poolConfig(bytes32)'s five return values.
// Field order must match the ABI's output order.
type poolConfigResult struct {
	Initialized      bool
	MemeIsCurrency0  bool
	Paused           bool
	Creator          common.Address
	PlatformTreasury common.Address
}

// Track reads poolConfig(poolId) - (initialized, memeIsCurrency0, paused, creator,
// platformTreasury) - and keeps the two fields the simulator needs.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hookTarget := hexutil.Encode(param.HookAddress[:])
	poolId := common.HexToHash(param.Pool.Address)

	// poolConfig returns five values; ethrpc decodes a multi-output method into a single
	// struct destination, not one pointer per output.
	var cfg poolConfigResult
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    premiumHookABI,
			Target: hookTarget,
			Method: "poolConfig",
			Params: []any{poolId},
		}, []any{&cfg}).
		Aggregate(); err != nil {
		return nil, err
	}
	return json.Marshal(Hook{
		MemeIsCurrency0: cfg.MemeIsCurrency0,
		Paused:          cfg.Paused,
	})
}

// BeforeSwap takes the 1% desk fee up front when desk is the specified (input) currency;
// otherwise returns a zero delta and defers to AfterSwap.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.Paused {
		return nil, ErrPoolPaused
	}
	if !params.CalcOut {
		return nil, ErrExactOutputDisabled
	}

	deltaSpecified := bignumber.ZeroBI
	if h.deskIsSpecified(params.ZeroForOne) {
		deltaSpecified = deskFee(params.AmountSpecified)
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: bignumber.ZeroBI,
		Gas:              gasBeforeSwap,
	}, nil
}

// AfterSwap takes the 1% desk fee out of the realized output when desk is the unspecified
// (output) currency; a no-op (fee already taken in BeforeSwap) otherwise.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if h.deskIsSpecified(params.ZeroForOne) {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI, Gas: gasAfterSwap}, nil
	}
	return &uniswapv4.AfterSwapResult{HookFee: deskFee(params.AmountOut), Gas: gasAfterSwap}, nil
}

// deskFee is floor(amount * 1%), PremiumLaunchHook's fee on whichever leg desk is specified/output on.
func deskFee(amount *big.Int) *big.Int {
	return bignumber.MulDivDown(new(big.Int), amount, big.NewInt(feeBps), big.NewInt(bps))
}

// deskIsSpecified mirrors PremiumLaunchHook._deskCurrencyAndSpecified: currency0 is the
// swap's specified (input) side iff zeroForOne, and desk is whichever currency meme is not.
func (h *Hook) deskIsSpecified(zeroForOne bool) bool {
	return zeroForOne != h.MemeIsCurrency0
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}
