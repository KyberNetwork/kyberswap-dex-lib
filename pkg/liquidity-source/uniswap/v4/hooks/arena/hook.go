package arena

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"golang.org/x/sync/singleflight"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook
	totalFeePpm *uint256.Int
}

type Extra struct {
	TotalFeePpm int64 `json:"f"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Arena},
	}

	var extra Extra
	if err := param.HookExtra.Unmarshal(&extra); err == nil {
		hook.totalFeePpm = uint256.NewInt(uint64(extra.TotalFeePpm))
	}
	return hook
}, HookAddresses...)

var (
	cachedHelperAddress string
	sf                  singleflight.Group
	cacheTime           = 30 * time.Minute
	timer               = time.AfterFunc(0, func() { cachedHelperAddress = "" })
)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
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
			return nil, err
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
		return nil, err
	}
	return json.Marshal(extra)
}

var MaxHookFee = uint256.NewInt(1e6)

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return &uniswapv4.AfterSwapResult{
		HookFee: u256.MulDivDown(new(uint256.Int),
			lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn), h.totalFeePpm, MaxHookFee),
	}, nil
}
