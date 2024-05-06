package slipstream

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	v3constants "github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	entities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func randomBigInt() *big.Int {
	words := make([]big.Word, 4)
	for i := range words {
		words[i] = big.Word(rand.Uint64())
	}
	return new(big.Int).SetBits(words)
}

func randomUint256() *uint256.Int {
	words := [4]uint64{}
	for i := range words {
		words[i] = rand.Uint64()
	}
	n := uint256.Int(words)
	return &n
}

func randomAddress() common.Address {
	buf := make([]byte, common.AddressLength)
	for i := range buf {
		buf[i] = byte(rand.Uint64() % 256)
	}
	return common.BytesToAddress(buf)
}

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolSimulator
	{
		ticksProvider, err := v3entities.NewTickListDataProvider([]v3entities.Tick{
			{
				Index:          v3utils.MinTick + 1,
				LiquidityNet:   int256.NewInt(10),
				LiquidityGross: uint256.NewInt(10),
			},
			{
				Index:          0,
				LiquidityNet:   int256.NewInt(-5),
				LiquidityGross: uint256.NewInt(5),
			},
			{
				Index:          v3utils.MaxTick - 1,
				LiquidityNet:   int256.NewInt(-5),
				LiquidityGross: uint256.NewInt(5),
			},
		}, 1)
		require.NoError(t, err)
		v3Pool := &v3entities.Pool{
			Token0:           entities.NewToken(uint(valueobject.ChainIDEthereum), randomAddress(), 18, "Token0", "Token0"),
			Token1:           entities.NewToken(uint(valueobject.ChainIDEthereum), randomAddress(), 18, "Token1", "Token1"),
			Fee:              v3constants.FeeAmount(rand.Uint64()),
			SqrtRatioX96:     randomUint256(),
			Liquidity:        randomUint256(),
			TickCurrent:      rand.Int(),
			TickDataProvider: ticksProvider,
		}
		pool := &PoolSimulator{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Address:    randomAddress().Hex(),
					ReserveUsd: rand.Float64(),
					SwapFee:    randomBigInt(),
					Exchange:   "uniswapv3",
					Type:       "uniswapv3",
					Tokens: []string{
						randomAddress().Hex(),
						randomAddress().Hex(),
					},
					Reserves: []*big.Int{
						randomBigInt(),
						randomBigInt(),
					},
					Checked:     true,
					BlockNumber: rand.Uint64(),
				},
			},
			V3Pool: v3Pool,
			gas: Gas{
				BaseGas:          int64(rand.Int()),
				CrossInitTickGas: int64(rand.Int()),
			},
			tickMin: rand.Int(),
			tickMax: rand.Int(),
		}
		pools = append(pools, pool)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolSimulator{})...))
	}
}
