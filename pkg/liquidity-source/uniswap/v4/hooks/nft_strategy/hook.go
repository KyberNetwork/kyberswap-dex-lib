package nftstrat

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	BlockTime = 12
)

type Hook struct {
	uniswapv4.Hook `json:"-"`
	DeploymentTime int64 `json:"dT,omitempty"`
	Fee            int64 `json:"f,omitempty"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4NftStrategy}}
	_ = param.HookExtra.Unmarshal(&hook)
	return hook
}, HookAddresses...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param.HookExtra != nil || param.HookAddress == PunkHookAddress {
		return json.RawMessage(param.HookExtra), nil
	}

	var deploymentBlock int64
	hookAddr := hexutil.Encode(param.HookAddress[:])
	resp, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hookAddr,
		Method: "deploymentBlock",
		Params: []any{common.HexToAddress(param.Pool.Tokens[1].Address)},
	}, []any{&deploymentBlock}).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hookAddr,
		Method: "fee",
	}, []any{&h.Fee}).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hookAddr,
		Method: "calculateFee",
		Params: []any{true},
	}, []any{&h.Fee}).TryBlockAndAggregate()
	if err != nil {
		return json.RawMessage("{}"), nil
	}

	if resp.BlockNumber != nil && deploymentBlock > 0 {
		h.DeploymentTime = time.Now().Unix() - BlockTime*(resp.BlockNumber.Int64()-deploymentBlock)
	}
	return json.Marshal(h)
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	fee := big.NewInt(h.calculateFee(params.ZeroForOne))
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.MulDivDown(fee,
			lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn), fee, bignumber.BasisPoint),
	}, nil
}

var (
	DefaultFee     int64 = 1000 // 10%
	StartingBuyFee int64 = 9500 // 95%
)

func (h *Hook) calculateFee(isBuying bool) int64 {
	if h.Fee > 0 {
		return h.Fee
	} else if !isBuying || h.DeploymentTime == 0 {
		return DefaultFee
	}

	blocksPassed := (time.Now().Unix() - h.DeploymentTime) / BlockTime
	feeReductions := (blocksPassed / 5) * 100 // bips to subtract

	maxReducible := StartingBuyFee - DefaultFee // assumes invariant holds
	if feeReductions >= maxReducible {
		return DefaultFee
	}
	return StartingBuyFee - feeReductions
}
