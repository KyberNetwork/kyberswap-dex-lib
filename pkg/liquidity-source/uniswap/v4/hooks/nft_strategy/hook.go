package nftstrat

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

const (
	BlockTime = 12
)

type Hook struct {
	uniswapv4.Hook
	NftStrategyExtra
}

type NftStrategyExtra struct {
	DeploymentTime int64 `json:"dT,omitempty"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4NftStrategy},
	}
	if param.HookExtra != "" {
		_ = json.Unmarshal([]byte(param.HookExtra), &hook.NftStrategyExtra)
	}
	return hook
}, HookAddresses...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	if param.HookExtra != "" || param.HookAddress == PunkHookAddress {
		return param.HookExtra, nil
	}

	var deploymentBlock int64
	resp, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "deploymentBlock",
		Params: []any{common.HexToAddress(param.Pool.Tokens[1].Address)},
	}, []any{&deploymentBlock}).TryBlockAndAggregate()
	if err != nil || resp.BlockNumber == nil {
		return "", err
	} else if deploymentBlock == 0 {
		return "{}", nil
	}

	extraBytes, _ := json.Marshal(NftStrategyExtra{
		DeploymentTime: time.Now().Unix() - BlockTime*(resp.BlockNumber.Int64()-deploymentBlock),
	})
	return string(extraBytes), nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	currentFee := big.NewInt(calculateFee(h.NftStrategyExtra, params.ZeroForOne))
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.MulDivDown(currentFee, params.AmountOut, currentFee, bignumber.BasisPoint),
	}, nil
}

var (
	DefaultFee     int64 = 1000 // 10%
	StartingBuyFee int64 = 9500 // 95%
)

func calculateFee(extra NftStrategyExtra, isBuying bool) int64 {
	if !isBuying || extra.DeploymentTime == 0 {
		return DefaultFee
	}

	blocksPassed := (time.Now().Unix() - extra.DeploymentTime) / BlockTime
	feeReductions := (blocksPassed / 5) * 100 // bips to subtract

	maxReducible := StartingBuyFee - DefaultFee // assumes invariant holds
	if feeReductions >= maxReducible {
		return DefaultFee
	}
	return StartingBuyFee - feeReductions
}
