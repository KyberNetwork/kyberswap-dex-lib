package deli

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// ConstantProductHook of idle takes 3% fee of ETH token
type ConstantProductHook struct {
	*uniswapv4.BaseHook
	Reserves [2]*uint256.Int
	FeeTier  *uint256.Int
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &ConstantProductHook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Deli},
	}
	if pool := param.Pool; pool != nil {
		hook.Reserves = [2]*uint256.Int{big256.New(pool.Reserves[0]), big256.New(pool.Reserves[1])}
		hook.FeeTier = uint256.NewInt(uint64(pool.SwapFee))
	}
	return hook
}, ConstantProductAddresses...)

func (h *ConstantProductHook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves,
	error) {
	var reserves [2]*big.Int
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    constantProductABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "getReserves",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&reserves}).Call(); err != nil {
		return nil, err
	}
	return entity.PoolReserves{reserves[0].String(), reserves[1].String()}, nil
}

func (h *ConstantProductHook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	amtSpecified := big256.FromBig(params.AmountSpecified)
	reserveIn, reserveOut := h.Reserves[1], h.Reserves[0]
	if params.ZeroForOne {
		reserveIn, reserveOut = reserveOut, reserveIn
	}
	var numerator, denominator uint256.Int
	amountInWithFee := numerator.Mul(amtSpecified, numerator.Sub(UFeeDenom, h.FeeTier))
	denominator.Add(denominator.Mul(reserveIn, UFeeDenom), amountInWithFee)
	numerator.Mul(amountInWithFee, reserveOut)
	amountOut := numerator.Div(&numerator, &denominator)
	amtUnspecified := amountOut.ToBig()
	amtUnspecified.Neg(amtUnspecified)

	swapInfo := [2]*uint256.Int{amountOut.Neg(amountOut), amtSpecified}
	if params.ZeroForOne {
		swapInfo[0], swapInfo[1] = swapInfo[1], swapInfo[0]
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   params.AmountSpecified,
		DeltaUnspecified: amtUnspecified,
		SwapInfo:         swapInfo,
	}, nil
}

func (h *ConstantProductHook) CloneState() uniswapv4.Hook {
	return lo.ToPtr(*h)
}

func (h *ConstantProductHook) UpdateBalance(swapInfoAny any) {
	swapInfo := swapInfoAny.([2]*uint256.Int)
	h.Reserves[0] = swapInfo[0].Add(h.Reserves[0], swapInfo[0])
	h.Reserves[1] = swapInfo[1].Add(h.Reserves[1], swapInfo[1])
}
