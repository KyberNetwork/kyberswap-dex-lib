package ekubov3

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
)

func TestDecodeVe33QuoteData(t *testing.T) {
	t.Parallel()

	type abiTick struct {
		Number         int32
		LiquidityDelta *big.Int
	}
	type abiQuoteData struct {
		Tick      int32
		SqrtRatio *big.Int
		Liquidity *big.Int
		MinTick   int32
		MaxTick   int32
		Ticks     []abiTick
	}
	type abiVe33QuoteData struct {
		QuoteData abiQuoteData
		SwapFee   uint64
	}

	expected := []abiVe33QuoteData{{
		QuoteData: abiQuoteData{
			Tick:      7,
			SqrtRatio: big.NewInt(11),
			Liquidity: big.NewInt(13),
			MinTick:   -17,
			MaxTick:   19,
			Ticks: []abiTick{{
				Number:         23,
				LiquidityDelta: big.NewInt(-29),
			}},
		},
		SwapFee: 31,
	}}

	method := abis.Ve33DataFetcherABI.Methods[ve33DataFetcherMethod]
	encoded, err := method.Outputs.Pack(expected)
	require.NoError(t, err)

	var decoded []ve33QuoteData
	require.NoError(t, abis.Ve33DataFetcherABI.UnpackIntoInterface(
		&decoded, ve33DataFetcherMethod, encoded,
	))
	require.Len(t, decoded, 1)
	require.Equal(t, expected[0].SwapFee, decoded[0].SwapFee)
	require.Equal(t, expected[0].QuoteData.Tick, decoded[0].QuoteData.Tick)
	require.Equal(t, expected[0].QuoteData.SqrtRatio, decoded[0].QuoteData.SqrtRatioFloat)
	require.Equal(t, expected[0].QuoteData.Ticks[0].LiquidityDelta, decoded[0].QuoteData.Ticks[0].LiquidityDelta)
}
