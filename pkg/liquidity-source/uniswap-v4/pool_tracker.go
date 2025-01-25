package uniswapv4

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var liquidity *big.Int

	var slot0 struct {
		SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
		// Tick         int64    `json:"tick"`
		// ProtocolFee  *big.Int `json:"protocolFee"`
		// LpFee        *big.Int `json:"lpFee"`
	}

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    stateViewABI,
		Target: staticExtra.StateViewAddress.Hex(),
		Method: "getLiquidity",
		Params: []interface{}{staticExtra.PoolId},
	}, []interface{}{&liquidity})

	calls.AddCall(&ethrpc.Call{
		ABI:    stateViewABI,
		Target: staticExtra.StateViewAddress.Hex(),
		Method: "getSlot0",
		Params: []interface{}{staticExtra.PoolId},
	}, []interface{}{&slot0})

	if _, err := calls.Aggregate(); err != nil {
		return p, nil
	}

	// reserve0 = liquidity * Q192 / sqrtPriceX96 / Q96
	reserve0 := new(big.Int).Mul(liquidity, Q192)
	reserve0.Div(reserve0, slot0.SqrtPriceX96)
	reserve0.Div(reserve0, Q96)

	// reserve1 = liquidity * sqrtPriceX96 / Q96
	reserve1 := new(big.Int).Mul(liquidity, slot0.SqrtPriceX96)
	reserve1.Div(reserve1, Q96)

	p.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}

	return p, nil
}
