package uniswapv2

import (
	"encoding/binary"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

type (
	IFeeTracker interface {
		AddFeeCall(request *ethrpc.Request, factoryAddress, poolAddress string, feeOutput *uint64)
	}

	// GenericFeeTracker gets fee generically, using {pool} and {factory} as templates for input common.Hash params
	GenericFeeTracker struct {
		abi    abi.ABI
		target string
		args   []string
	}
)

func NewGenericFeeTracker(feeTrackerCfg *FeeTrackerCfg) *GenericFeeTracker {
	return &GenericFeeTracker{
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

func getGenericInput(input, poolAddress, factoryAddress string) string {
	switch input {
	case genericTemplatePool:
		return poolAddress
	case genericTemplateFactory:
		return factoryAddress
	default:
		return input
	}
}

// AddFeeCall appends the fee read to request so it can be batched with other pool reads.
func (t *GenericFeeTracker) AddFeeCall(request *ethrpc.Request,
	factoryAddress, poolAddress string, feeOutput *uint64) {
	request.AddCall(&ethrpc.Call{
		ABI:    t.abi,
		Target: getGenericInput(t.target, poolAddress, factoryAddress),
		Method: genericMethodFee,
		Params: lo.Map(t.args, func(arg string, _ int) any {
			return common.HexToHash(getGenericInput(arg, poolAddress, factoryAddress))
		}),
	}, []any{feeOutput})
}
