package alpha

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = cl.RegisterHooksFactory(func(param *cl.HookParam) cl.Hook {
	hook := &Hook{Hook: cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLTax, param)}
	if len(param.HookExtra) > 0 {
		_ = json.Unmarshal(param.HookExtra, &hook.Extra)
	}
	return hook
},
	common.HexToAddress("0x26F251A46D15f396d2095738ad19869a13d4c9fD"),
)

type Hook struct {
	cl.Hook
	Extra
}

type Extra struct {
	TaxRate        *big.Int `json:"t,omitempty"`
	TaxCurrencyIdx int      `json:"c,omitempty"`
}

func (h *Hook) Track(ctx context.Context, param *cl.HookParam) ([]byte, error) {
	if len(param.HookExtra) > 0 {
		return param.HookExtra, nil
	}

	var extra Extra
	var taxCurrency common.Address
	hook := hexutil.Encode(param.HookAddress[:])
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hook,
		Method: "TAX_RATE",
	}, []any{&extra.TaxRate}).AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hook,
		Method: "taxCurrency",
	}, []any{&taxCurrency}).TryAggregate(); err != nil {
		return nil, err
	}
	extra.TaxCurrencyIdx = lo.Ternary(common.HexToAddress(param.Pool.Tokens[0].Address) == taxCurrency, 0, 1)
	return json.Marshal(extra)
}

func (h *Hook) BeforeSwap(params *cl.BeforeSwapParams) (*cl.BeforeSwapResult, error) {
	deltaSpecific := bignumber.ZeroBI
	if params.ZeroForOne == (h.TaxCurrencyIdx == 0) {
		deltaSpecific = bignumber.MulDivDown(new(big.Int), params.AmountSpecified, h.TaxRate, bignumber.BasisPoint)
	}
	return &cl.BeforeSwapResult{
		DeltaSpecified:   deltaSpecific,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

func (h *Hook) AfterSwap(params *cl.AfterSwapParams) (*cl.AfterSwapResult, error) {
	hookFeeAmt := bignumber.ZeroBI
	if params.ZeroForOne != (h.TaxCurrencyIdx == 0) {
		hookFeeAmt = bignumber.MulDivDown(new(big.Int), params.AmountOut, h.TaxRate, bignumber.BasisPoint)
	}
	return &cl.AfterSwapResult{
		HookFee: hookFeeAmt,
	}, nil
}
