package tickerdualpool_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
	ticker "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/ticker-dualpool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestTickerDualPoolSerializationPreservesQuotes(t *testing.T) {
	address := common.HexToAddress("0x047ea2f55148b7601fdde0d28588b55c09192ac4")
	backing := uint256.MustFromDecimal("1000000000000000000000000")
	price := new(uint256.Int).Lsh(uint256.NewInt(1), 96)
	snapshot := ticker.Snapshot{BlockNumber: 123, Pool: dualpool.Extra{Live: true, Balance0: backing.Clone(), Balance1: backing.Clone(), SqrtPriceX96: price, LpFee: 2000, Buckets: []dualpool.Bucket{{TickLower: -10000, TickUpper: 10000, WeightBps: 10000}}}, Total0: backing.Clone(), Total1: backing.Clone(), TickSpacing: 10, Registered: true, HookFeeBps: 100, CreatorTaxBps: 50}
	shares := uint256.MustFromDecimal("1000000000000000000000000000000")
	snapshot.Aave[0] = &ticker.AaveInventory{Raw: new(uint256.Int), Claims: new(uint256.Int), PoolShares: shares.Clone(), Supply: shares.Clone(), Idle: new(uint256.Int), StrategyShares: backing.Clone(), Rate: uint256.MustFromDecimal("1000000000000000000000000000"), PendingYield: new(uint256.Int)}
	snapshot.Vaults[0] = common.HexToAddress("0x3")
	snapshot.Reserves[0] = &ticker.AaveReserve{Cash: backing.Clone(), ScaledSupply: backing.Clone(), SupplyCap: new(uint256.Int), PendingTreasury: new(uint256.Int), Active: true}
	raw, err := json.Marshal(snapshot)
	require.NoError(t, err)
	fetch := v4.FetchRPCResult{Liquidity: big.NewInt(0), TickSpacing: 10, Slot0: v4.Slot0Data{SqrtPriceX96: price.ToBig(), Tick: big.NewInt(0), LpFee: 2000}, HookExtra: raw}
	extra, err := json.Marshal(fetch.ToExtra([]v4.Tick{}))
	require.NoError(t, err)
	static, err := json.Marshal(v4.StaticExtra{Fee: 2000, TickSpacing: 10, HooksAddress: address})
	require.NoError(t, err)
	ep := entity.Pool{Address: "0x1", Type: v4.DexType, Tokens: []*entity.PoolToken{{Address: "0x0000000000000000000000000000000000000001", Decimals: 18}, {Address: "0x0000000000000000000000000000000000000002", Decimals: 18}}, Reserves: entity.PoolReserves{backing.Dec(), backing.Dec()}, Extra: string(extra), StaticExtra: string(static), BlockNumber: 123}
	original, err := v4.NewPoolSimulator(ep, valueobject.ChainIDMonad)
	require.NoError(t, err)
	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{ep.Address: original})
	require.NoError(t, err)
	decoded, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	restored := decoded[ep.Address]
	input := pool.TokenAmount{Token: ep.Tokens[0].Address, Amount: big.NewInt(1001)}
	params := pool.CalcAmountOutParams{TokenAmountIn: input, TokenOut: ep.Tokens[1].Address}
	for range 2 {
		a, err := original.CalcAmountOut(params)
		require.NoError(t, err)
		b, err := restored.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, a.TokenAmountOut.Amount.String(), b.TokenAmountOut.Amount.String())
		original.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: input, TokenAmountOut: *a.TokenAmountOut, SwapInfo: a.SwapInfo})
		restored.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: input, TokenAmountOut: *b.TokenAmountOut, SwapInfo: b.SwapInfo})
	}
}
