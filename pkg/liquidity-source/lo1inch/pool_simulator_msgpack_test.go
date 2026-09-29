package lo1inch_test

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch"
	helper1inch "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Router-service ships pool simulators over msgpack, so the full-fill-only rule has to survive the round-trip.
func TestPoolSimulator_NoPartialFillsSurvivesMsgpack(t *testing.T) {
	t.Parallel()

	const (
		takerAsset = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		makerAsset = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	)

	makerTraits := "0x" + helper1inch.DefaultMakerTraits().DisablePartialFills().Build().Text(16)

	extra, err := json.Marshal(lo1inch.Extra{TakeToken0Orders: []*lo1inch.Order{{
		OrderHash:            "0x177af74e4d3880743ac6603323a9a50f6999968e499f44966dd00d642e933285",
		Maker:                "0xdf4039a454d58868dfd43f076ee46c92a35fdfd9",
		MakerAsset:           makerAsset,
		TakerAsset:           takerAsset,
		MakingAmount:         uint256.NewInt(10000),
		TakingAmount:         uint256.NewInt(101),
		RemainingMakerAmount: uint256.NewInt(10000),
		MakerBalance:         uint256.NewInt(10000),
		MakerAllowance:       uint256.NewInt(10000),
		MakerTraits:          makerTraits,
	}}})
	require.NoError(t, err)

	sim, err := lo1inch.NewPoolSimulator(entity.Pool{
		Address:     "lo1inch_" + takerAsset + "_" + makerAsset,
		Tokens:      []*entity.PoolToken{{Address: takerAsset}, {Address: makerAsset}},
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: `{"token0":"` + takerAsset + `","token1":"` + makerAsset + `"}`,
		Extra:       string(extra),
	})
	require.NoError(t, err)

	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{"pool": sim})
	require.NoError(t, err)
	decoded, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)

	_, err = decoded["pool"].CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: takerAsset, Amount: big.NewInt(50)},
		TokenOut:      makerAsset,
	})
	require.ErrorIs(t, err, lo1inch.ErrCannotFulfillAmountIn)
}
