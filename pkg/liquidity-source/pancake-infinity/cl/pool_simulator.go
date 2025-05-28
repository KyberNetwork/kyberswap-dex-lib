package cl

import (
	"fmt"
	"strings"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	defaultGas = uniswapv3.Gas{BaseGas: 75000, CrossInitTickGas: 21000}
)

type PoolSimulator struct {
	*uniswapv3.PoolSimulator
	staticExtra StaticExtra
	hook        Hook
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshal static extra: %w", err)
	}

	hook, ok := GetHook(staticExtra.HooksAddress)
	if !ok && staticExtra.HasSwapPermissions {
		return nil, shared.ErrUnsupportedHook
	}

	v3PoolSimulator, err := uniswapv3.NewPoolSimulator(entityPool, chainID)
	if err != nil {
		return nil, err
	}
	if entityPool.Tokens[0].Address > entityPool.Tokens[1].Address {
		// restore original order after V3Pool constructor forced sorting
		v3Pool := v3PoolSimulator.V3Pool
		v3Pool.Token0, v3Pool.Token1 = v3Pool.Token1, v3Pool.Token0
	}
	v3PoolSimulator.Gas = defaultGas

	return &PoolSimulator{
		PoolSimulator: v3PoolSimulator,
		staticExtra:   staticExtra,
		hook:          hook,
	}, nil
}

func (p *PoolSimulator) GetExchange() string {
	return p.hook.GetExchange()
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*uniswapv3.PoolSimulator)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInAddress, tokenOutAddress := eth.AddressZero, eth.AddressZero
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenIn)] {
		tokenInAddress = common.HexToAddress(tokenIn)
	}
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenOut)] {
		tokenOutAddress = common.HexToAddress(tokenOut)
	}

	zeroForOne := strings.EqualFold(tokenIn, p.GetTokens()[0])
	var priceLimit v3Utils.Uint160
	_ = p.GetSqrtPriceLimit(zeroForOne, &priceLimit)

	return PoolMetaInfo{
		Vault:       p.staticExtra.VaultAddress,
		PoolManager: p.staticExtra.PoolManagerAddress,
		Permit2Addr: p.staticExtra.Permit2Address,
		TokenIn:     tokenInAddress,
		TokenOut:    tokenOutAddress,
		Fee:         p.staticExtra.Fee,
		Parameters:  p.staticExtra.Parameters,
		HookAddress: p.staticExtra.HooksAddress,
		HookData:    []byte{},
		PriceLimit:  &priceLimit,
	}
}
