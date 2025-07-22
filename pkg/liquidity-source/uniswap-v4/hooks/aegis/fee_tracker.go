package aegis

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type DynamicFeeStateRPC struct {
	BaseFee  *big.Int
	SurgeFee *big.Int
}
type ManualFeeRPC struct {
	ManualFee *big.Int
	IsSet     bool
}

func TrackFee(ctx context.Context, hookAddress common.Address, poolId string, ethrpcClient *ethrpc.Client) func(pool *entity.Pool) error {
	return func(pool *entity.Pool) error {
		var staticExtra StaticExtraAegis
		if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err != nil {
			return err
		}

		if !shared.IsDynamicFee(staticExtra.Fee) {
			return errors.New("not a dynamic fee")
		}
		if staticExtra.DynamicFeeManagerAddress == (common.Address{}) {
			req := ethrpcClient.NewRequest().SetContext(ctx)
			req.AddCall(&ethrpc.Call{
				ABI:    aegisHookABI,
				Target: pool.Address,
				Method: "policyManager",
			}, []any{&staticExtra.PolicyManagerAddress})
			req.AddCall(&ethrpc.Call{
				ABI:    aegisHookABI,
				Target: pool.Address,
				Method: "dynamicFeeManager",
			}, []any{&staticExtra.DynamicFeeManagerAddress})
			_, err := req.Aggregate()
			if err != nil {
				return err
			}
			staticExtraBytes, err := json.Marshal(staticExtra)
			if err != nil {
				return err
			}
			pool.StaticExtra = string(staticExtraBytes)
		}
		var extra ExtraAegis
		if err := json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
			return err
		}
		req := ethrpcClient.NewRequest().SetContext(ctx)
		var dynamicFeeState DynamicFeeStateRPC
		var manualFee ManualFeeRPC
		var poolPOLShare *big.Int
		req.AddCall(&ethrpc.Call{
			ABI:    aegisDynamicFeeManagerABI,
			Target: staticExtra.DynamicFeeManagerAddress.Hex(),
			Method: "getFeeState",
			Params: []any{eth.StringToBytes32(pool.Address)},
		}, []any{&dynamicFeeState})
		req.AddCall(&ethrpc.Call{
			ABI:    aegisPoolPolicyManagerABI,
			Target: staticExtra.PolicyManagerAddress.Hex(),
			Method: "getManualFee",
			Params: []any{eth.StringToBytes32(pool.Address)},
		}, []any{&manualFee})

		req.AddCall(&ethrpc.Call{
			ABI:    aegisPoolPolicyManagerABI,
			Target: staticExtra.PolicyManagerAddress.Hex(),
			Method: "getPoolPOLShare",
			Params: []any{eth.StringToBytes32(pool.Address)},
		}, []any{&poolPOLShare})

		extra.BaseFee = dynamicFeeState.BaseFee.Uint64()
		extra.SurgeFee = dynamicFeeState.SurgeFee.Uint64()
		extra.ManualFee = manualFee.ManualFee.Uint64()
		extra.ManualFeeIsSet = manualFee.IsSet
		extra.DynamicFee = lo.Ternary(extra.ManualFeeIsSet, extra.ManualFee, extra.BaseFee+extra.SurgeFee)
		extra.PoolPOLShare = poolPOLShare.Uint64()

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return err
		}
		pool.Extra = string(extraBytes)
		pool.SwapFee = float64(extra.DynamicFee)
		return nil
	}
}
