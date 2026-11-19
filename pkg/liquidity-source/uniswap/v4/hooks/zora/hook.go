package zora

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Fee constants of ZoraV4CoinHook._calculateLaunchFee (CoinConstants.sol), in pips.
const (
	lpFee         uniswapv4.FeeAmount = 10_000
	trendLpFee    uniswapv4.FeeAmount = 100
	launchFee     uniswapv4.FeeAmount = 990_000
	launchSeconds int64               = 10
)

// trendHook is the only hook version that charges trendLpFee (v2.6.0).
var trendHook = common.HexToAddress("0x0469a4Bd3724DC86C9542F4694c976DA13C450c0")

// Hook is a ZoraV4CoinHook. Hooks with beforeSwap return a dynamic fee from the coin's creation time and type.
// A zero CreatedAt (legacy coin, or untracked pool) means the flat lpFee.
type Hook struct {
	uniswapv4.Hook `json:"-"`
	CreatedAt      int64 `json:"c,omitempty"`
	Trend          bool  `json:"t,omitempty"`
	hookAddress    common.Address
	quoteAt        int64
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook:        &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Zora},
		hookAddress: param.HookAddress,
		quoteAt:     param.QuoteAt,
	}
	_ = param.HookExtra.Unmarshal(hook)
	return hook
}, HookAddresses...)

// Track reads the coin's creation time and type once; both are immutable.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if len(param.HookExtra) > 0 || !h.CanBeforeSwap(param.HookAddress) {
		return json.RawMessage(param.HookExtra), nil
	}

	var poolCoin struct {
		Offset *big.Int
		Coin   common.Address
	}
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "getPoolCoinByHash",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&poolCoin}).Aggregate(); err != nil {
		return nil, err
	}

	// A legacy coin has no creationInfo/coinType and reverts: it pays the flat fee.
	var info struct {
		CreationTimestamp *big.Int
		IsDeploying       bool
	}
	var coinType uint8
	coin := hexutil.Encode(poolCoin.Coin[:])
	res, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
		ABI:    coinABI,
		Target: coin,
		Method: "creationInfo",
	}, []any{&info}).AddCall(&ethrpc.Call{
		ABI:    coinABI,
		Target: coin,
		Method: "coinType",
	}, []any{&coinType}).TryAggregate()
	if err != nil {
		return nil, err
	}
	h.CreatedAt, h.Trend = 0, false
	if res.Result[0] && info.CreationTimestamp != nil && !info.IsDeploying {
		h.CreatedAt = info.CreationTimestamp.Int64()
		h.Trend = res.Result[1] && coinType == coinTypeTrend
	}
	return json.Marshal(h)
}

const coinTypeTrend = 2

func (h *Hook) BeforeSwap(_ *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	now := h.quoteAt
	if now == 0 {
		now = time.Now().Unix()
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          uniswapv4.OverrideFeeFlag | h.fee(now),
	}, nil
}

// fee is ZoraV4CoinHook._calculateLaunchFee: the launch fee decays linearly from 99% to the LP fee
// over launchSeconds after creation, then the LP fee applies.
func (h *Hook) fee(now int64) uniswapv4.FeeAmount {
	flat := lpFee
	if h.Trend && h.hookAddress == trendHook {
		flat = trendLpFee
	}
	if h.CreatedAt == 0 {
		return lpFee
	}
	if elapsed := now - h.CreatedAt; elapsed >= launchSeconds {
		return flat
	} else if elapsed >= 0 {
		return launchFee - uniswapv4.FeeAmount(elapsed*int64(launchFee-lpFee)/launchSeconds)
	}
	return flat
}
