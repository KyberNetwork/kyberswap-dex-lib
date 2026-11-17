package velodromev1

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolTracker_AddBalanceCalls(t *testing.T) {
	t.Parallel()
	d := &PoolTracker{}
	p := entity.Pool{
		Address: "0x421a018cc5839c4c0300afb21c725776dc389b1a",
		Tokens: []*entity.PoolToken{
			{Address: "0x1111111111111111111111111111111111111111"},
			{Address: "0x2222222222222222222222222222222222222222"},
		},
	}

	req := ethrpc.New("http://127.0.0.1:0").NewRequest()
	var bals [2]*big.Int
	d.addBalanceCalls(req, p, bals[:])
	require.Len(t, req.Calls, 2)
	for i, call := range req.Calls {
		assert.Equal(t, p.Tokens[i].Address, call.Target)
		assert.Equal(t, "balanceOf", call.Method)
		assert.Len(t, call.Params, 1)
	}

	req = ethrpc.New("http://127.0.0.1:0").NewRequest()
	p.Tokens = p.Tokens[:1]
	d.addBalanceCalls(req, p, bals[:])
	assert.Empty(t, req.Calls)
}

type fakeLogDecoder struct {
	reserves ReserveData
	block    *big.Int
}

func (f fakeLogDecoder) Decode([]ethtypes.Log, map[uint64]entity.BlockHeader) (ReserveData, *big.Int, error) {
	return f.reserves, f.block, nil
}

// TestPoolTracker_GetNewPoolState runs GetNewPoolState against a stubbed multicall: the reserves, the factory
// pause flag and the balanceOf(pair) of both tokens come from one aggregate at one block. The pool it writes is then
// quoted by the simulator, e.g. polygon dystopia USD+/USDC.e 0x421a018cc5839c4c0300afb21c725776dc389b1a: reserves
// report 999.5M USD+ but the pair holds none.
func TestPoolTracker_GetNewPoolState(t *testing.T) {
	t.Parallel()
	const (
		pairAddr = "0x421a018cc5839c4c0300afb21c725776dc389b1a"
		token0   = "0x236eec6359fb44cce8f97e99387aa7f8cd5cde1f"
		token1   = "0x2791bca1f2de4661ed88a30c99a7a9449aa84174"
		factory  = "0x1d21Db6cde1b18c7E47B0F7F42f4b3F68b9beeC9"
		block    = uint64(100)
	)
	reserve0, reserve1 := big.NewInt(999_500_000_000_000), big.NewInt(730_000)

	// balanceOf(pair) of each token, nil = the call fails inside the multicall
	newTracker := func(t *testing.T, bal0, bal1 *big.Int) (*PoolTracker, *testutil.MulticallStub) {
		stub := testutil.NewMulticallStub(t, func(target common.Address, data []byte) (bool, []byte) {
			switch {
			case target == common.HexToAddress(factory):
				ret, err := pairFactoryABI.Methods[pairFactoryMethodIsPaused].Outputs.Pack(false)
				require.NoError(t, err)
				return true, ret
			case target == common.HexToAddress(pairAddr):
				ret, err := pairABI.Methods[pairMethodGetReserves].Outputs.Pack(reserve0, reserve1, uint32(1))
				require.NoError(t, err)
				return true, ret
			}
			bal := map[common.Address]*big.Int{common.HexToAddress(token0): bal0, common.HexToAddress(token1): bal1}[target]
			if bal == nil {
				return false, nil
			}
			return true, common.LeftPadBytes(bal.Bytes(), 32)
		}).At(block, 1_700_000_000)
		d, err := NewPoolTracker(&Config{FactoryAddress: factory, Fee: 5}, stub.Client())
		require.NoError(t, err)
		return d, stub
	}
	pool := entity.Pool{
		Address:  pairAddr,
		Exchange: "dystopia",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true, Decimals: 6},
			{Address: token1, Swappable: true, Decimals: 6},
		},
		Reserves:    entity.PoolReserves{"1", "1"},
		StaticExtra: `{"feePrecision":10000,"stable":true}`,
	}
	onChainReserves := entity.PoolReserves{reserve0.String(), reserve1.String()}
	quote := func(p entity.Pool, tokenIn int) error {
		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		amount := big.NewInt(200_000) // 0.2 USDC.e
		if tokenIn == 0 {
			amount = big.NewInt(1_000_000_000) // 1000 USD+: the curve is so skewed that less gives a zero amount out
		}
		_, err = sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{Token: p.Tokens[tokenIn].Address, Amount: amount},
			TokenOut:      p.Tokens[1-tokenIn].Address,
		})
		return err
	}
	extraOf := func(p entity.Pool) PoolExtra {
		var extra PoolExtra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		return extra
	}

	t.Run("RPC path", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			bal0, bal1 *big.Int
			wantBals   bool
		}{
			{"balances equal reserves", reserve0, reserve1, true},
			{"balance above reserve", new(big.Int).Add(reserve0, big.NewInt(5)), reserve1, true},
			{"token0 drained", big.NewInt(0), reserve1, true},
			{"token1 short by 1", reserve0, new(big.Int).Sub(reserve1, big.NewInt(1)), true},
			{"balanceOf of one token fails: balances unknown", nil, reserve1, false},
			{"balanceOf of both tokens fail: balances unknown", nil, nil, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				d, stub := newTracker(t, tc.bal0, tc.bal1)
				got, err := d.GetNewPoolState(context.Background(), pool, poolpkg.GetNewPoolStateParams{})
				require.NoError(t, err)
				// the reserves are the on-chain ones in every case
				assert.Equal(t, onChainReserves, got.Reserves)
				assert.Equal(t, block, got.BlockNumber)
				extra := extraOf(got)
				if tc.wantBals {
					assert.Equal(t, tc.bal0, extra.Balance0)
					assert.Equal(t, tc.bal1, extra.Balance1)
				} else {
					assert.Nil(t, extra.Balance0)
					assert.Nil(t, extra.Balance1)
				}
				// one aggregate: isPaused, getReserves and the 2 balanceOf
				require.Len(t, stub.Targets(), 1)
				assert.Len(t, stub.Targets()[0], 4)
			})
		}
	})

	t.Run("drained pool is not quoted, with its reserves kept, and recovers", func(t *testing.T) {
		d, _ := newTracker(t, big.NewInt(0), reserve1)
		drained, err := d.GetNewPoolState(context.Background(), pool, poolpkg.GetNewPoolStateParams{})
		require.NoError(t, err)
		assert.Equal(t, onChainReserves, drained.Reserves)
		assert.ErrorIs(t, quote(drained, 1), ErrInsufficientBalance)     // USDC.e in, USD+ out
		assert.ErrorIs(t, quote(drained, 0), ErrInsufficientInputAmount) // USD+ in, USDC.e out

		d, _ = newTracker(t, reserve0, reserve1)
		recovered, err := d.GetNewPoolState(context.Background(), drained, poolpkg.GetNewPoolStateParams{})
		require.NoError(t, err)
		assert.NoError(t, quote(recovered, 1))
		assert.NoError(t, quote(recovered, 0))

		// a pool without balances (balanceOf failed) is quoted as before
		d, _ = newTracker(t, nil, nil)
		unknown, err := d.GetNewPoolState(context.Background(), pool, poolpkg.GetNewPoolStateParams{})
		require.NoError(t, err)
		assert.NoError(t, quote(unknown, 1))
	})

	t.Run("log path reads the balances at the block of the Sync log", func(t *testing.T) {
		d, stub := newTracker(t, big.NewInt(0), reserve1)
		d.logDecoder = fakeLogDecoder{
			reserves: uniswapv2.ReserveData{Reserve0: reserve0, Reserve1: reserve1, BlockTimestampLast: 1},
			block:    new(big.Int).SetUint64(block),
		}
		got, err := d.GetNewPoolState(context.Background(), pool,
			poolpkg.GetNewPoolStateParams{Logs: []ethtypes.Log{{}}})
		require.NoError(t, err)
		assert.Equal(t, onChainReserves, got.Reserves)
		assert.Zero(t, extraOf(got).Balance0.Sign())
		assert.Equal(t, reserve1, extraOf(got).Balance1)
		require.Len(t, stub.Targets(), 1)
		assert.Len(t, stub.Targets()[0], 3) // isPaused and the 2 balanceOf, the reserves come from the log

		// the aggregate is pinned to the log's block: a stub chain at another block rejects it
		d.logDecoder = fakeLogDecoder{
			reserves: uniswapv2.ReserveData{Reserve0: reserve0, Reserve1: reserve1},
			block:    new(big.Int).SetUint64(block - 1),
		}
		_, err = d.GetNewPoolState(context.Background(), pool, poolpkg.GetNewPoolStateParams{Logs: []ethtypes.Log{{}}})
		assert.Error(t, err)
	})
}
