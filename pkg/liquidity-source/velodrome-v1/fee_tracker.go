package velodromev1

import (
	"context"
	"encoding/binary"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

type (
	IFeeTracker interface {
		GetFee(ctx context.Context, factoryAddress, poolAddress string, isStable bool,
			blockNumber *big.Int) (uint64, error)
		AddGetFeeCall(request *ethrpc.Request, factoryAddress, poolAddress string, isStable bool,
			feeOutput *uint64) *ethrpc.Request
	}

	// GenericFeeTracker gets fee generically, using {pool} and {factory} as templates for input common.Hash params
	GenericFeeTracker struct {
		ethrpcClient *ethrpc.Client
		abi          abi.ABI
		target       string
		args         []string
	}
)

func NewGenericFeeTracker(ethrpcClient *ethrpc.Client, feeTrackerCfg *FeeTrackerCfg) IFeeTracker {
	if feeTrackerCfg == nil {
		return nil
	}
	return &GenericFeeTracker{
		ethrpcClient: ethrpcClient,
		abi: abi.ABI{
			Methods: map[string]abi.Method{
				genericMethodFee: {
					ID: binary.BigEndian.AppendUint32(make([]byte, 0, 4), feeTrackerCfg.Selector),
					Inputs: lo.RepeatBy(len(feeTrackerCfg.Args), func(int) abi.Argument {
						return abi.Argument{Type: abi.Type{T: abi.FixedBytesTy, Size: 32}}
					}),
					Outputs: abi.Arguments{
						{Type: abi.Type{T: abi.UintTy, Size: 64}},
					},
				},
			},
		},
		target: feeTrackerCfg.Target,
		args:   feeTrackerCfg.Args,
	}
}

func getGenericInput(input, poolAddress, factoryAddress string, isStable bool) string {
	switch input {
	case genericTemplatePool:
		return poolAddress
	case genericTemplateFactory:
		return factoryAddress
	case genericTemplateIsStable:
		if isStable {
			return "01"
		}
		return ""
	default:
		return input
	}
}

func (t *GenericFeeTracker) GetFee(ctx context.Context, factoryAddress, poolAddress string, isStable bool,
	blockNumber *big.Int) (fee uint64, err error) {
	_, err = t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber).AddCall(&ethrpc.Call{
		ABI:    t.abi,
		Target: getGenericInput(t.target, poolAddress, factoryAddress, isStable),
		Method: genericMethodFee,
		Params: lo.Map(t.args, func(arg string, _ int) any {
			return common.HexToHash(getGenericInput(arg, poolAddress, factoryAddress, isStable))
		}),
	}, []any{&fee}).Call()
	return fee, err
}

func (t *GenericFeeTracker) AddGetFeeCall(request *ethrpc.Request,
	factoryAddress, poolAddress string, isStable bool, feeOutput *uint64) *ethrpc.Request {
	return request.AddCall(&ethrpc.Call{
		ABI:    t.abi,
		Target: getGenericInput(t.target, poolAddress, factoryAddress, isStable),
		Method: genericMethodFee,
		Params: lo.Map(t.args, func(arg string, _ int) any {
			return common.HexToHash(getGenericInput(arg, poolAddress, factoryAddress, isStable))
		}),
	}, []any{feeOutput})
}
