package angstrom

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := L2Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4AngstromL2}}
	_ = param.HookExtra.Unmarshal(&h)
	return &h
}, L2HookAddresses...)

type L2Hook struct {
	uniswapv4.Hook `json:"-"`
	SwapFeeE6      int64 `json:"f"`
}

func (h *L2Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	var staticExtra uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(param.Pool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	currency0, currency1 := uniswapv4.NativeTokenAddress, uniswapv4.NativeTokenAddress
	if !staticExtra.IsNative[0] {
		currency0 = common.HexToAddress(param.Pool.Tokens[0].Address)
	}
	if !staticExtra.IsNative[1] {
		currency1 = common.HexToAddress(param.Pool.Tokens[1].Address)
	}

	var feeConfig struct {
		FeeConfig struct {
			IsInitialized     bool
			CreatorTaxFeeE6   uint32
			ProtocolTaxFeeE6  uint32
			CreatorSwapFeeE6  uint32
			ProtocolSwapFeeE6 uint32
		}
	}
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "getPoolFeeConfiguration",
		Params: []any{uniswapv4.PoolKey{
			Currency0:   currency0,
			Currency1:   currency1,
			Fee:         big.NewInt(int64(staticExtra.Fee)),
			TickSpacing: big.NewInt(int64(staticExtra.TickSpacing)),
			Hooks:       staticExtra.HooksAddress,
		}},
	}, []any{&feeConfig}).Call(); err != nil {
		return nil, err
	}

	h.SwapFeeE6 = int64(feeConfig.FeeConfig.CreatorSwapFeeE6 + feeConfig.FeeConfig.ProtocolSwapFeeE6)

	return json.Marshal(h)
}

var uFactorE6 = uint256.NewInt(1_000_000)

func (h *L2Hook) BeforeSwap(param *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	deltaSpecified := new(uint256.Int)
	if param.ZeroForOne {
		fee := uint256.NewInt(uint64(h.SwapFeeE6))
		u256.MulDivDown(deltaSpecified, param.AmountSpecified, fee, uFactorE6)
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: new(uint256.Int),
	}, nil
}

func (h *L2Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	hookFeeAmt := new(uint256.Int)
	if !params.ZeroForOne {
		fee := uint256.NewInt(uint64(h.SwapFeeE6))
		u256.MulDivDown(hookFeeAmt, lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn),
			fee, uFactorE6)
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFeeAmt,
	}, nil
}
