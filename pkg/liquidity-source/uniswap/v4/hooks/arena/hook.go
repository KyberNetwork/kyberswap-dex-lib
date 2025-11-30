package arena

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"golang.org/x/sync/singleflight"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook
	totalFeePpm *big.Int
}

type Extra struct {
	TotalFeePpm int64 `json:"f"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Arena},
	}

	if param.HookExtra != "" {
		var extra Extra
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err == nil {
			hook.totalFeePpm = big.NewInt(extra.TotalFeePpm)
		}
	}
	return hook
}, HookAddresses...)

var (
	cachedHelperAddress string
	sf                  singleflight.Group
	cacheTime           = 30 * time.Minute
	timer               = time.AfterFunc(0, func() { cachedHelperAddress = "" })
)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	helper := cachedHelperAddress
	if helper == "" {
		res, err, _ := sf.Do("", func() (any, error) {
			var arenaFeeHelper common.Address
			_, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
				ABI:    hookABI,
				Target: hexutil.Encode(param.HookAddress[:]),
				Method: "arenaFeeHelper",
			}, []any{&arenaFeeHelper}).Call()
			helper = hexutil.Encode(arenaFeeHelper[:])
			cachedHelperAddress = helper
			timer.Reset(cacheTime)
			return helper, err
		})
		if err != nil {
			return "", err
		}
		helper = res.(string)
	}

	var extra Extra
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: helper,
		Method: "getTotalFeePpm",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&extra.TotalFeePpm}).Call(); err != nil {
		return "", err
	}
	extraBytes, _ := json.Marshal(extra)
	return string(extraBytes), nil
}

var MaxHookFee = bignumber.TenPowInt(6)

func (h *Hook) AfterSwap(swapHookParams *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	feeAmt := new(big.Int)
	feeAmt.Mul(swapHookParams.AmountOut, h.totalFeePpm).Div(feeAmt, MaxHookFee)
	return &uniswapv4.AfterSwapResult{
		HookFee: feeAmt,
	}, nil
}
