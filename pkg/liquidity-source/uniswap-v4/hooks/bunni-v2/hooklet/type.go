package hooklet

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type IHooklet interface {
	Track(context.Context, HookletParams) (string, error)
	BeforeSwap(*SwapParams) (feeOverriden bool, fee *uint256.Int, priceOverridden bool, sqrtPriceX96 *uint256.Int)
	AfterSwap(*SwapParams)
}

type HookletParams struct {
	RpcClient      *ethrpc.Client
	HookletAddress common.Address
	HookletExtra   string
	PoolId         [32]byte
}

type SwapParams struct {
	ZeroForOne bool
}
