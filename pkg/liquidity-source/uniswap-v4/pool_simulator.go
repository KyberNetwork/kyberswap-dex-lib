package uniswapv4

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	defaultGas = uniswapv3.Gas{BaseGas: 75000, CrossInitTickGas: 21000}
)

type PoolSimulator struct {
	*uniswapv3.PoolSimulator

	IsNative [2]bool

	staticExtra StaticExtra
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshal static extra: %w", err)
	}

	if HasSwapPermissions(staticExtra.HooksAddress) {
		return nil, ErrUnsupportedHook
	}

	v3PoolSimulator, err := uniswapv3.NewPoolSimulator(entityPool, chainID)
	if err != nil {
		return nil, err
	}
	v3PoolSimulator.Gas = defaultGas
	return &PoolSimulator{
		PoolSimulator: v3PoolSimulator,
		IsNative: [2]bool{
			staticExtra.Currency0 == EmptyAddress,
			staticExtra.Currency1 == EmptyAddress,
		},
		staticExtra: staticExtra,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*uniswapv3.PoolSimulator)
	return &cloned
}

// GetMetaInfo
// adapt from https://github.com/KyberNetwork/kyberswap-dex-lib-private/blob/c1877a8c19759faeb7d82b6902ed335f0657ce3e/pkg/liquidity-source/uniswap-v4/pool_simulator.go#L201
func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	tokenInAddress, tokenOutAddress := NativeTokenAddress, NativeTokenAddress
	if !p.IsNative[p.GetTokenIndex(tokenIn)] {
		tokenInAddress = common.HexToAddress(tokenIn)
	} else if !p.IsNative[p.GetTokenIndex(tokenOut)] {
		tokenOutAddress = common.HexToAddress(tokenOut)
	}

	return PoolMetaInfo{
		Router:      p.staticExtra.UniversalRouterAddress,
		Permit2Addr: p.staticExtra.Permit2Address,
		TokenIn:     tokenInAddress,
		TokenOut:    tokenOutAddress,
		Fee:         p.staticExtra.Fee,
		TickSpacing: p.staticExtra.TickSpacing,
		HookAddress: p.staticExtra.HooksAddress,
		HookData:    []byte{},
	}
}
