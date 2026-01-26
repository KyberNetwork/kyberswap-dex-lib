package hooks

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const (
	HookBeforeSwap uint8 = 1 << 0 // 1
	HookGetFee     uint8 = 1 << 1 // 2
	HookAfterSwap  uint8 = 1 << 2 // 4
)

type GetFeeParams struct {
	Asset0IsInput bool
	Reserve0      *uint256.Int
	Reserve1      *uint256.Int
}

type BeforeSwapParams struct {
	AmountOut  *uint256.Int
	ZeroForOne bool
}

type AfterSwapParams struct {
	AmountIn   *uint256.Int
	AmountOut  *uint256.Int
	Fee        *uint256.Int
	ZeroForOne bool
	Reserve0   *uint256.Int
	Reserve1   *uint256.Int
}

type HookParam struct {
	RpcClient   *ethrpc.Client
	Pool        *entity.Pool
	HookAddress common.Address
	HookExtra   string
	BlockNumber *big.Int
}

type Hook interface {
	GetFee(params *GetFeeParams) (uint64, error)

	BeforeSwap(params *BeforeSwapParams) error

	AfterSwap(params *AfterSwapParams) error

	Track(ctx context.Context, param *HookParam) (string, error)

	CloneState() Hook
}

type HookFactory func(param *HookParam) Hook

var HookFactories = map[common.Address]HookFactory{}

func RegisterHooksFactory(factory HookFactory, addresses ...common.Address) bool {
	for _, address := range addresses {
		HookFactories[address] = factory
	}
	return true
}

func GetHook(hookAddress common.Address, param *HookParam) Hook {
	if param == nil {
		param = &HookParam{}
	}
	param.HookAddress = hookAddress

	factory, ok := HookFactories[hookAddress]
	if ok && factory != nil {
		return factory(param)
	}

	return NewBaseHook(param)
}
