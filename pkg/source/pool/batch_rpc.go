package pool

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	ethabi "github.com/KyberNetwork/ethrpc/abi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// IBatchRPCPoolTracker plans RPC calls without executing them so a worker can batch
// calls across pools and pool types before building entity.Pool values for storage.
type IBatchRPCPoolTracker interface {
	LazyNewPoolState(ctx context.Context, p entity.Pool, params GetNewPoolStateParams) (ILazyRequest, func(*big.Int) (entity.Pool, error), error)
}

type ILazyRequest interface {
	GetUnpacks() []func([]byte) error
	GetCallMsgs() []ethereum.CallMsg
	GetEthRpcCall(idx int) *ethrpc.Call
}

type LazyRequest struct {
	*ethrpc.Request
	Unpacks  []func([]byte) error
	CallMsgs []ethereum.CallMsg
}

func (r *LazyRequest) AddCall(c *ethrpc.Call, output []interface{}) *LazyRequest {
	r.Request.AddCall(c, output)
	target := common.HexToAddress(c.Target)
	data, _ := ethabi.Pack(&c.ABI, c.Method, c.Params...)

	r.CallMsgs = append(r.CallMsgs, ethereum.CallMsg{
		To:   &target,
		Data: data,
	})

	r.Unpacks = append(r.Unpacks, func(res []byte) error {
		err := ethabi.UnpackArgs(c.ABI.Methods[c.Method].Outputs, output[0], res)
		if err != nil {
			return err
		}
		return nil
	})
	return r
}

func (r *LazyRequest) GetUnpacks() []func([]byte) error {
	return r.Unpacks
}

func (r *LazyRequest) GetCallMsgs() []ethereum.CallMsg {
	return r.CallMsgs
}

func (r *LazyRequest) GetEthRpcCall(idx int) *ethrpc.Call {
	return r.Calls[idx]
}
