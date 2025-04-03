package eclp

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCalcAmountOut(t *testing.T) {
	testcases := []struct {
		name              string
		poolJSON          string
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut string
		expectedSwapFee   string
	}{
		{
			name:     "1. 0x5d7f2aac9999950f6ffb03394be584e1410bcfaf",
			poolJSON: "{\"address\":\"0x5d7f2aac9999950f6ffb03394be584e1410bcfaf\",\"exchange\":\"balancer-v3-eclp\",\"type\":\"balancer-v3-eclp\",\"timestamp\":1743666215,\"reserves\":[\"7112661012533552\",\"4570881\"],\"tokens\":[{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"symbol\":\"WETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"hook\\\":{},\\\"fee\\\":\\\"1000000000000000\\\",\\\"aggrFee\\\":\\\"500000000000000000\\\",\\\"balsE18\\\":[\\\"7416589241266276\\\",\\\"5143281691971181138\\\"],\\\"decs\\\":[\\\"1\\\",\\\"1000000000000\\\"],\\\"rates\\\":[\\\"1042730593823768440\\\",\\\"1125227651293302350\\\"],\\\"buffs\\\":[{\\\"tA\\\":\\\"981220933663162500476\\\",\\\"tS\\\":\\\"941010975869571861421\\\"},{\\\"tA\\\":\\\"16002487184920\\\",\\\"tS\\\":\\\"14221555227204\\\"}],\\\"eclp\\\":{\\\"p\\\":{\\\"a\\\":\\\"1550000000000000000000\\\",\\\"b\\\":\\\"2900000000000000000000\\\",\\\"c\\\":\\\"476190422200635\\\",\\\"s\\\":\\\"999999886621334475\\\",\\\"l\\\":\\\"6000000000000000000000\\\"},\\\"d\\\":{\\\"tA\\\":{\\\"x\\\":\\\"-71194417720710388791873272380661517967\\\",\\\"y\\\":\\\"70223606325857393780377068191710603749\\\"},\\\"tB\\\":{\\\"x\\\":\\\"61901682449602783283884409155259788043\\\",\\\"y\\\":\\\"78537772504117652540633453925274067422\\\"},\\\"u\\\":\\\"63379080947523002588928208779431795\\\",\\\"v\\\":\\\"78537770618819626952384805221876672653\\\",\\\"w\\\":\\\"3959125853791535734173172101662641\\\",\\\"z\\\":\\\"-71194387540195651900451013855438212676\\\",\\\"DSq\\\":\\\"100000000000000000034081090601792885000\\\"}}}\",\"staticExtra\":\"{\\\"buffs\\\":[\\\"0x0bfc9d54fc184518a81162f8fb99c2eaca081202\\\",\\\"0xd4fa2d31b7968e448877f69a96de69f5de8cd23e\\\"]}\",\"blockNumber\":22186972}",
			tokenAmountIn: poolpkg.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: big.NewInt(1e14),
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "182071", //182068
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var pool entity.Pool
			err := json.Unmarshal([]byte(tc.poolJSON), &pool)
			assert.Nil(t, err)
			s, err := NewPoolSimulator(pool)
			assert.Nil(t, err)
			result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
				})
			})

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount.String())
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	testcases := []struct {
		name             string
		poolJSON         string
		tokenAmountOut   poolpkg.TokenAmount
		tokenIn          string
		expectedAmountIn string
	}{
		{
			name:     "1. 0x5d7f2aac9999950f6ffb03394be584e1410bcfaf",
			poolJSON: "{\"address\":\"0x5d7f2aac9999950f6ffb03394be584e1410bcfaf\",\"exchange\":\"balancer-v3-eclp\",\"type\":\"balancer-v3-eclp\",\"timestamp\":1743666215,\"reserves\":[\"7112661012533552\",\"4570881\"],\"tokens\":[{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"symbol\":\"WETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"hook\\\":{},\\\"fee\\\":\\\"1000000000000000\\\",\\\"aggrFee\\\":\\\"500000000000000000\\\",\\\"balsE18\\\":[\\\"7416589241266276\\\",\\\"5143281691971181138\\\"],\\\"decs\\\":[\\\"1\\\",\\\"1000000000000\\\"],\\\"rates\\\":[\\\"1042730593823768440\\\",\\\"1125227651293302350\\\"],\\\"buffs\\\":[{\\\"tA\\\":\\\"981220933663162500476\\\",\\\"tS\\\":\\\"941010975869571861421\\\"},{\\\"tA\\\":\\\"16002487184920\\\",\\\"tS\\\":\\\"14221555227204\\\"}],\\\"eclp\\\":{\\\"p\\\":{\\\"a\\\":\\\"1550000000000000000000\\\",\\\"b\\\":\\\"2900000000000000000000\\\",\\\"c\\\":\\\"476190422200635\\\",\\\"s\\\":\\\"999999886621334475\\\",\\\"l\\\":\\\"6000000000000000000000\\\"},\\\"d\\\":{\\\"tA\\\":{\\\"x\\\":\\\"-71194417720710388791873272380661517967\\\",\\\"y\\\":\\\"70223606325857393780377068191710603749\\\"},\\\"tB\\\":{\\\"x\\\":\\\"61901682449602783283884409155259788043\\\",\\\"y\\\":\\\"78537772504117652540633453925274067422\\\"},\\\"u\\\":\\\"63379080947523002588928208779431795\\\",\\\"v\\\":\\\"78537770618819626952384805221876672653\\\",\\\"w\\\":\\\"3959125853791535734173172101662641\\\",\\\"z\\\":\\\"-71194387540195651900451013855438212676\\\",\\\"DSq\\\":\\\"100000000000000000034081090601792885000\\\"}}}\",\"staticExtra\":\"{\\\"buffs\\\":[\\\"0x0bfc9d54fc184518a81162f8fb99c2eaca081202\\\",\\\"0xd4fa2d31b7968e448877f69a96de69f5de8cd23e\\\"]}\",\"blockNumber\":22186972}",
			tokenAmountOut: poolpkg.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(182071),
			},
			tokenIn:          "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			expectedAmountIn: "99998114793038", //182068
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var pool entity.Pool
			err := json.Unmarshal([]byte(tc.poolJSON), &pool)
			assert.Nil(t, err)
			s, err := NewPoolSimulator(pool)
			assert.Nil(t, err)
			result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
				return s.CalcAmountIn(poolpkg.CalcAmountInParams{
					TokenAmountOut: tc.tokenAmountOut,
					TokenIn:        tc.tokenIn,
				})
			})

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedAmountIn, result.TokenAmountIn.Amount.String())
		})
	}
}
