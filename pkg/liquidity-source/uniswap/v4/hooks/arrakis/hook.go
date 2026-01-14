package arrakis

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct { // scheduled
	uniswapv4.Hook
	Extra
}

type Extra struct {
	FeesFrom [2]uniswapv4.FeeAmount `json:"fs"`
}

type FeesDataRPC struct {
	Module        common.Address
	ZeroForOneFee uint32
	OneForZeroFee uint32
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	var extra Extra
	if param.HookExtra != "" {
		_ = json.Unmarshal([]byte(param.HookExtra), &extra)
	}
	return &Hook{
		Hook:  &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Arrakis},
		Extra: extra,
	}
}, HookAddresses...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var feesData FeesDataRPC
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "getFeesData",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&feesData}).Call(); err != nil {
		return "", err
	}

	extraBytes, _ := json.Marshal(Extra{FeesFrom: [2]uniswapv4.FeeAmount{
		uniswapv4.FeeAmount(feesData.ZeroForOneFee),
		uniswapv4.FeeAmount(feesData.OneForZeroFee),
	}})
	return string(extraBytes), nil
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          h.Extra.FeesFrom[lo.Ternary(params.ZeroForOne, 0, 1)],
	}, nil
}
