package hooklet

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type feeOverrideHooklet struct {
	HookletExtra
}

type HookletExtra struct {
	OverrideZeroToOne bool
	FeeZeroToOne      *uint256.Int
	OverrideOneToZero bool
	FeeOneToZero      *uint256.Int
}

type FeeOverrideRPC struct {
	OverrideZeroToOne bool
	FeeZeroToOne      *big.Int
	OverrideOneToZero bool
	FeeOneToZero      *big.Int
}

func NewFeeOverrideHooklet(extra uniswapv4.HookExtra) IHooklet {
	var hookletExtra HookletExtra
	if extra != nil {
		if err := extra.Unmarshal(&hookletExtra); err != nil {
			return nil
		}
	}

	return &feeOverrideHooklet{
		HookletExtra: hookletExtra,
	}
}

func (h *feeOverrideHooklet) Track(ctx context.Context, params HookletParams) (json.RawMessage, error) {
	var feeOverride FeeOverrideRPC

	req := params.RpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    feeOverrideHookletABI,
		Target: params.HookletAddress.Hex(),
		Method: "feeOverrides",
		Params: []any{params.PoolId},
	}, []any{&feeOverride})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return json.Marshal(&HookletExtra{
		OverrideZeroToOne: feeOverride.OverrideZeroToOne,
		FeeZeroToOne:      uint256.MustFromBig(feeOverride.FeeZeroToOne),
		OverrideOneToZero: feeOverride.OverrideOneToZero,
		FeeOneToZero:      uint256.MustFromBig(feeOverride.FeeOneToZero),
	})
}

func (h *feeOverrideHooklet) BeforeSwap(params *SwapParams) (
	feeOverriden bool,
	fee *uint256.Int,
	priceOverridden bool,
	sqrtPriceX96 *uint256.Int,
) {
	if params.ZeroForOne {
		return h.OverrideZeroToOne, h.FeeZeroToOne, false, u256.U0
	}

	return h.OverrideOneToZero, h.FeeOneToZero, false, u256.U0
}

func (h *feeOverrideHooklet) AfterSwap(_ *SwapParams) {}

func (h *feeOverrideHooklet) CloneState() IHooklet {
	cloned := *h
	cloned.FeeOneToZero = h.FeeOneToZero.Clone()
	cloned.FeeZeroToOne = h.FeeZeroToOne.Clone()
	return &cloned
}
