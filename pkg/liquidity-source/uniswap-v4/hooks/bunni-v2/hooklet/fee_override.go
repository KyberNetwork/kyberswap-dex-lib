package hooklet

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"
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

func NewFeeOverrideHooklet(extra string) IHooklet {
	var hookletExtra HookletExtra
	if extra != "" {
		if err := json.Unmarshal([]byte(extra), &hookletExtra); err != nil {
			return nil
		}
	}

	return &feeOverrideHooklet{
		HookletExtra: hookletExtra,
	}
}

func (h *feeOverrideHooklet) Track(ctx context.Context, params HookletParams) (string, error) {
	var feeOverride FeeOverrideRPC

	req := params.RpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    feeOverrideHookletABI,
		Target: params.HookletAddress.Hex(),
		Method: "feeOverrides",
		Params: []any{params.PoolId},
	}, []any{&feeOverride})

	if _, err := req.Aggregate(); err != nil {
		return "", err
	}

	extra, err := json.Marshal(&HookletExtra{
		OverrideZeroToOne: feeOverride.OverrideZeroToOne,
		FeeZeroToOne:      uint256.MustFromBig(feeOverride.FeeZeroToOne),
		OverrideOneToZero: feeOverride.OverrideOneToZero,
		FeeOneToZero:      uint256.MustFromBig(feeOverride.FeeOneToZero),
	})

	if err != nil {
		return "", err
	}

	return string(extra), nil
}

func (h *feeOverrideHooklet) BeforeSwap(params *SwapParams) (bool, *uint256.Int, bool, *uint256.Int) {
	if params.ZeroForOne {
		return h.OverrideZeroToOne, h.FeeZeroToOne, false, new(uint256.Int)
	}

	return h.OverrideOneToZero, h.FeeOneToZero, false, new(uint256.Int)
}

func (h *feeOverrideHooklet) AfterSwap(_ *SwapParams) {}

func (h *feeOverrideHooklet) CloneState() IHooklet {
	cloned := *h
	cloned.FeeOneToZero = h.FeeOneToZero.Clone()
	cloned.FeeZeroToOne = h.FeeZeroToOne.Clone()
	return &cloned
}
