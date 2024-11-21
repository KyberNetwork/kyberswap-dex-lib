package mxtrading

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

var entityPoolStrData = "{\"address\":\"mx_trading_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2_0xfaba6f8e4a5e8ab82f62fe7c39859fa577269be3\",\"" +
	"exchange\":\"mx-trading\",\"" + "type\":\"mx-trading\",\"" + "timestamp\":1732581492,\"" +
	"reserves\":[\"59925038314246815744\",\"16488225768595991298048\"],\"" +
	"tokens\":[{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"" +
	"symbol\":\"WETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xfaba6f8e4a5e8ab82f62fe7c39859fa577269be3\",\"symbol\":\"ONDO\",\"decimals\":18,\"swappable\":true}],\"" +
	"extra\":\"{\\\"0to1\\\":[{\\\"s\\\":0.719,\\\"p\\\":3347.4385889037885},{\\\"s\\\":0.015,\\\"p\\\":3347.141106167435},{\\\"s\\\":0.012,\\\"p\\\":3347.131414506469},{\\\"s\\\":0.015,\\\"p\\\":3347.1120311845366},{\\\"s\\\":0.012,\\\"p\\\":3346.9507374280724},{\\\"s\\\":0.434,\\\"p\\\":3346.768097038609},{\\\"s\\\":0.015,\\\"p\\\":3346.7584063173954},{\\\"s\\\":0.006,\\\"p\\\":3346.7487155961812},{\\\"s\\\":0.006,\\\"p\\\":3346.7390248749675},{\\\"s\\\":0.021,\\\"p\\\":3346.729334153753},{\\\"s\\\":0.012,\\\"p\\\":3346.709952711325},{\\\"s\\\":0.015,\\\"p\\\":3346.700261990111},{\\\"s\\\":0.012,\\\"p\\\":3346.680880547683},{\\\"s\\\":0.021,\\\"p\\\":3346.6711898264693},{\\\"s\\\":0.033,\\\"p\\\":3346.6421176628264},{\\\"s\\\":0.027,\\\"p\\\":3346.6130454991844},{\\\"s\\\":0.59,\\\"p\\\":3346.4124246485567},{\\\"s\\\":2.9644734252274776,\\\"p\\\":3343.4441414468833}],\\\"" +
	"1to0\\\":[{\\\"s\\\":546.1,\\\"p\\\":0.00029818453400477844},{\\\"s\\\":85.3,\\\"p\\\":0.0002981556065179715},{\\\"s\\\":879.2,\\\"p\\\":0.0002981266790311648},{\\\"s\\\":2262.7,\\\"p\\\":0.000298097751544358},{\\\"s\\\":6897.1,\\\"p\\\":0.00029806882405755117},{\\\"s\\\":776.9,\\\"p\\\":0.00029803989657074435},{\\\"s\\\":1680.8,\\\"p\\\":0.0002980109690839375},{\\\"s\\\":705.5,\\\"p\\\":0.00029798204159713065},{\\\"s\\\":3007.2,\\\"p\\\":0.00029795311411032384},{\\\"s\\\":2726.6,\\\"p\\\":0.00029792418662351696},{\\\"s\\\":2439,\\\"p\\\":0.0002978952591367102},{\\\"s\\\":2804.8,\\\"p\\\":0.0002978663316499034},{\\\"s\\\":3993.1,\\\"p\\\":0.0002978374041630965},{\\\"s\\\":10061.9,\\\"p\\\":0.00029780847667628974},{\\\"s\\\":7804.2,\\\"p\\\":0.0002977795491894829},{\\\"s\\\":2159.1,\\\"p\\\":0.00029775062170267605},{\\\"s\\\":4587.7,\\\"p\\\":0.0002977216942158692},{\\\"s\\\":2491.2,\\\"p\\\":0.0002976927667290623},{\\\"s\\\":2613.2,\\\"p\\\":0.0002976638392422556},{\\\"s\\\":9751,\\\"p\\\":0.0002976349117554487},{\\\"s\\\":3181.1,\\\"p\\\":0.0002976059842686419},{\\\"s\\\":5084.1,\\\"p\\\":0.0002975770567818351},{\\\"s\\\":6101.5,\\\"p\\\":0.00029754812929502826},{\\\"s\\\":38407,\\\"p\\\":0.00029751920180822133},{\\\"s\\\":31967.7,\\\"p\\\":0.0002974902743214145},{\\\"s\\\":22651.1,\\\"p\\\":0.00029746134683460775},{\\\"s\\\":2021.2,\\\"p\\\":0.00029743241934780093},{\\\"s\\\":3600.3,\\\"p\\\":0.00029740349186099417},{\\\"s\\\":6000.3,\\\"p\\\":0.0002973745643741873},{\\\"s\\\":3594.4,\\\"p\\\":0.0002973456368873804},{\\\"s\\\":4878.2,\\\"p\\\":0.00029731670940057365},{\\\"s\\\":5603.656713348088,\\\"p\\\":0.00029728778191376684}]}\"}"

var entityPoolData = entity.Pool{
	Address:  "mx_trading_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2_0xfaba6f8e4a5e8ab82f62fe7c39859fa577269be3",
	Exchange: "mx-trading",
	Type:     "mx-trading",
	Reserves: []string{"59925038314246815744", "16488225768595991298048"},
	Tokens: []*entity.PoolToken{
		{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Symbol: "WETH", Decimals: 18, Swappable: true},
		{Address: "0xfaba6f8e4a5e8ab82f62fe7c39859fa577269be3", Symbol: "ONDO", Decimals: 18, Swappable: true},
	},
	Extra: "{\"0to1\":[{\"s\":0.719,\"p\":3347.4385889037885},{\"s\":0.015,\"p\":3347.141106167435},{\"s\":0.012,\"p\":3347.131414506469},{\"s\":0.015,\"p\":3347.1120311845366},{\"s\":0.012,\"p\":3346.9507374280724},{\"s\":0.434,\"p\":3346.768097038609},{\"s\":0.015,\"p\":3346.7584063173954},{\"s\":0.006,\"p\":3346.7487155961812},{\"s\":0.006,\"p\":3346.7390248749675},{\"s\":0.021,\"p\":3346.729334153753},{\"s\":0.012,\"p\":3346.709952711325},{\"s\":0.015,\"p\":3346.700261990111},{\"s\":0.012,\"p\":3346.680880547683},{\"s\":0.021,\"p\":3346.6711898264693},{\"s\":0.033,\"p\":3346.6421176628264},{\"s\":0.027,\"p\":3346.6130454991844},{\"s\":0.59,\"p\":3346.4124246485567},{\"s\":2.9644734252274776,\"p\":3343.4441414468833}],\"" +
		"1to0\":[{\"s\":546.1,\"p\":0.00029818453400477844},{\"s\":85.3,\"p\":0.0002981556065179715},{\"s\":879.2,\"p\":0.0002981266790311648},{\"s\":2262.7,\"p\":0.000298097751544358},{\"s\":6897.1,\"p\":0.00029806882405755117},{\"s\":776.9,\"p\":0.00029803989657074435},{\"s\":1680.8,\"p\":0.0002980109690839375},{\"s\":705.5,\"p\":0.00029798204159713065},{\"s\":3007.2,\"p\":0.00029795311411032384},{\"s\":2726.6,\"p\":0.00029792418662351696},{\"s\":2439,\"p\":0.0002978952591367102},{\"s\":2804.8,\"p\":0.0002978663316499034},{\"s\":3993.1,\"p\":0.0002978374041630965},{\"s\":10061.9,\"p\":0.00029780847667628974},{\"s\":7804.2,\"p\":0.0002977795491894829},{\"s\":2159.1,\"p\":0.00029775062170267605},{\"s\":4587.7,\"p\":0.0002977216942158692},{\"s\":2491.2,\"p\":0.0002976927667290623},{\"s\":2613.2,\"p\":0.0002976638392422556},{\"s\":9751,\"p\":0.0002976349117554487},{\"s\":3181.1,\"p\":0.0002976059842686419},{\"s\":5084.1,\"p\":0.0002975770567818351},{\"s\":6101.5,\"p\":0.00029754812929502826},{\"s\":38407,\"p\":0.00029751920180822133},{\"s\":31967.7,\"p\":0.0002974902743214145},{\"s\":22651.1,\"p\":0.00029746134683460775},{\"s\":2021.2,\"p\":0.00029743241934780093},{\"s\":3600.3,\"p\":0.00029740349186099417},{\"s\":6000.3,\"p\":0.0002973745643741873},{\"s\":3594.4,\"p\":0.0002973456368873804},{\"s\":4878.2,\"p\":0.00029731670940057365},{\"s\":5603.656713348088,\"p\":0.00029728778191376684}]}",
}

func TestNewPoolSimulator(t *testing.T) {
	entityPool := entity.Pool{}
	err := json.Unmarshal([]byte(entityPoolStrData), &entityPool)
	assert.NoError(t, err)
	reflect.DeepEqual(entityPoolData, entityPool)

	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)
	assert.NotNil(t, poolSimulator.OneToZeroPriceLevels)
	assert.NotNil(t, poolSimulator.ZeroToOnePriceLevels)

	checkPriceLevels := func(levels []PriceLevel, decimals uint8, quoteTokenReserve *big.Int) {
		reserveF := 0.
		totalBaseSizeF := 0.
		for _, level := range levels {
			assert.Greater(t, level.Size, 0.)
			assert.Greater(t, level.Price, 0.)
			reserveF += level.Size * level.Price
			totalBaseSizeF += level.Size * math.Pow10(int(decimals))
		}
		totalBaseSize, _ := big.NewFloat(totalBaseSizeF).Int(nil)
		fmt.Println("totalBaseSize: " + totalBaseSize.String())
		reserve, _ := big.NewFloat(reserveF * math.Pow10(int(decimals))).Int(nil)
		assert.Equal(t, reserve.Uint64(), quoteTokenReserve.Uint64())
	}

	checkPriceLevels(poolSimulator.ZeroToOnePriceLevels, poolSimulator.token1.Decimals, poolSimulator.GetReserves()[1])
	checkPriceLevels(poolSimulator.OneToZeroPriceLevels, poolSimulator.token0.Decimals, poolSimulator.GetReserves()[0])
}

func TestPoolSimulator_GetAmountOut(t *testing.T) {
	tests := []struct {
		name                 string
		amountIn0, amountIn1 *big.Int
		expectedAmountOut    *big.Int
		expectedErr          error
	}{
		{
			name:        "it should return error when amountIn0 higher than total level size",
			amountIn0:   big.NewInt(5_000000000_000000000),
			expectedErr: ErrAmountInIsGreaterThanTotalLevelSize,
		},
		{
			name:        "it should return error when amountIn1 higher than total level size",
			amountIn1:   bignumber.NewBig("300000000000000000000000"),
			expectedErr: ErrAmountInIsGreaterThanTotalLevelSize,
		},
		{
			name:              "it should return correct amountOut1 when all levels are filled",
			amountIn0:         bignumber.NewBig("4929473425227476992"),
			expectedAmountOut: bignumber.NewBig("16488225768595991298048"),
		},
		{
			name:              "it should return correct amountOut0 when all levels are filled",
			amountIn1:         bignumber.NewBig("201363156713348041539584"),
			expectedAmountOut: bignumber.NewBig("59925038314246815744"),
		},
		{
			name: "it should return correct amountOut",
			// 0.719 + 0.01 = 0.729
			amountIn0: bignumber.NewBig("729000000000000000"),
			// 0.729 * (0.719 * 3347.4385889037885 / 0.729 + 0.01 * 3347.141106167435 / 0.729)
			expectedAmountOut: bignumber.NewBig("2440279756483498344448"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolSimulator, err := NewPoolSimulator(entityPoolData)
			assert.NoError(t, err)
			entityPool := entity.Pool{}
			_ = json.Unmarshal([]byte(entityPoolStrData), &entityPool)

			tokenIn, tokenOut, amountIn := entityPool.Tokens[0].Address, entityPool.Tokens[1].Address, tt.amountIn0
			if amountIn == nil {
				tokenIn, tokenOut, amountIn = tokenOut, tokenIn, tt.amountIn1
			}
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
				TokenOut:      tokenOut,
				Limit:         swaplimit.NewInventory("mx-trading", poolSimulator.CalculateLimit()),
			}

			result, err := poolSimulator.CalcAmountOut(params)
			if assert.Equal(t, tt.expectedErr, err) && tt.expectedErr == nil {
				assert.Equal(t, tt.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	tests := []struct {
		name                         string
		amountIn0, amountIn1         *big.Int
		expectedZeroToOnePriceLevels []PriceLevel
		expectedOneToZeroPriceLevels []PriceLevel
		expectedErr                  error
	}{
		{
			name:      "fill token0",
			amountIn0: bignumber.NewBig("10000000000000000"),
			expectedZeroToOnePriceLevels: []PriceLevel{
				{Size: 0.709, Price: 3347.4385889037885},
				{Size: 0.015, Price: 3347.141106167435},
				{Size: 0.012, Price: 3347.131414506469},
				{Size: 0.015, Price: 3347.1120311845366},
				{Size: 0.012, Price: 3346.9507374280724},
				{Size: 0.434, Price: 3346.768097038609},
				{Size: 0.015, Price: 3346.7584063173954},
				{Size: 0.006, Price: 3346.7487155961812},
				{Size: 0.006, Price: 3346.7390248749675},
				{Size: 0.021, Price: 3346.729334153753},
				{Size: 0.012, Price: 3346.709952711325},
				{Size: 0.015, Price: 3346.700261990111},
				{Size: 0.012, Price: 3346.680880547683},
				{Size: 0.021, Price: 3346.6711898264693},
				{Size: 0.033, Price: 3346.6421176628264},
				{Size: 0.027, Price: 3346.6130454991844},
				{Size: 0.59, Price: 3346.4124246485567},
				{Size: 2.9644734252274776, Price: 3343.4441414468833},
			},
			expectedOneToZeroPriceLevels: []PriceLevel{
				{Size: 546.1, Price: 0.00029818453400477844},
				{Size: 85.3, Price: 0.0002981556065179715},
				{Size: 879.2, Price: 0.0002981266790311648},
				{Size: 2262.7, Price: 0.000298097751544358},
				{Size: 6897.1, Price: 0.00029806882405755117},
				{Size: 776.9, Price: 0.00029803989657074435},
				{Size: 1680.8, Price: 0.0002980109690839375},
				{Size: 705.5, Price: 0.00029798204159713065},
				{Size: 3007.2, Price: 0.00029795311411032384},
				{Size: 2726.6, Price: 0.00029792418662351696},
				{Size: 2439, Price: 0.0002978952591367102},
				{Size: 2804.8, Price: 0.0002978663316499034},
				{Size: 3993.1, Price: 0.0002978374041630965},
				{Size: 10061.9, Price: 0.00029780847667628974},
				{Size: 7804.2, Price: 0.0002977795491894829},
				{Size: 2159.1, Price: 0.00029775062170267605},
				{Size: 4587.7, Price: 0.0002977216942158692},
				{Size: 2491.2, Price: 0.0002976927667290623},
				{Size: 2613.2, Price: 0.0002976638392422556},
				{Size: 9751, Price: 0.0002976349117554487},
				{Size: 3181.1, Price: 0.0002976059842686419},
				{Size: 5084.1, Price: 0.0002975770567818351},
				{Size: 6101.5, Price: 0.00029754812929502826},
				{Size: 38407, Price: 0.00029751920180822133},
				{Size: 31967.7, Price: 0.0002974902743214145},
				{Size: 22651.1, Price: 0.00029746134683460775},
				{Size: 2021.2, Price: 0.00029743241934780093},
				{Size: 3600.3, Price: 0.00029740349186099417},
				{Size: 6000.3, Price: 0.0002973745643741873},
				{Size: 3594.4, Price: 0.0002973456368873804},
				{Size: 4878.2, Price: 0.00029731670940057365},
				{Size: 5603.656713348088, Price: 0.00029728778191376684},
			},
		},
		{
			name:                         "fill all levels 0to1",
			amountIn0:                    bignumber.NewBig("4929473425227476992"),
			expectedZeroToOnePriceLevels: nil,
			expectedOneToZeroPriceLevels: []PriceLevel{
				{Size: 546.1, Price: 0.00029818453400477844},
				{Size: 85.3, Price: 0.0002981556065179715},
				{Size: 879.2, Price: 0.0002981266790311648},
				{Size: 2262.7, Price: 0.000298097751544358},
				{Size: 6897.1, Price: 0.00029806882405755117},
				{Size: 776.9, Price: 0.00029803989657074435},
				{Size: 1680.8, Price: 0.0002980109690839375},
				{Size: 705.5, Price: 0.00029798204159713065},
				{Size: 3007.2, Price: 0.00029795311411032384},
				{Size: 2726.6, Price: 0.00029792418662351696},
				{Size: 2439, Price: 0.0002978952591367102},
				{Size: 2804.8, Price: 0.0002978663316499034},
				{Size: 3993.1, Price: 0.0002978374041630965},
				{Size: 10061.9, Price: 0.00029780847667628974},
				{Size: 7804.2, Price: 0.0002977795491894829},
				{Size: 2159.1, Price: 0.00029775062170267605},
				{Size: 4587.7, Price: 0.0002977216942158692},
				{Size: 2491.2, Price: 0.0002976927667290623},
				{Size: 2613.2, Price: 0.0002976638392422556},
				{Size: 9751, Price: 0.0002976349117554487},
				{Size: 3181.1, Price: 0.0002976059842686419},
				{Size: 5084.1, Price: 0.0002975770567818351},
				{Size: 6101.5, Price: 0.00029754812929502826},
				{Size: 38407, Price: 0.00029751920180822133},
				{Size: 31967.7, Price: 0.0002974902743214145},
				{Size: 22651.1, Price: 0.00029746134683460775},
				{Size: 2021.2, Price: 0.00029743241934780093},
				{Size: 3600.3, Price: 0.00029740349186099417},
				{Size: 6000.3, Price: 0.0002973745643741873},
				{Size: 3594.4, Price: 0.0002973456368873804},
				{Size: 4878.2, Price: 0.00029731670940057365},
				{Size: 5603.656713348088, Price: 0.00029728778191376684},
			},
		},
		{
			name:      "fill token1",
			amountIn1: bignumber.NewBig("201363000000000000000000"),
			expectedZeroToOnePriceLevels: []PriceLevel{
				{Size: 0.719, Price: 3347.4385889037885},
				{Size: 0.015, Price: 3347.141106167435},
				{Size: 0.012, Price: 3347.131414506469},
				{Size: 0.015, Price: 3347.1120311845366},
				{Size: 0.012, Price: 3346.9507374280724},
				{Size: 0.434, Price: 3346.768097038609},
				{Size: 0.015, Price: 3346.7584063173954},
				{Size: 0.006, Price: 3346.7487155961812},
				{Size: 0.006, Price: 3346.7390248749675},
				{Size: 0.021, Price: 3346.729334153753},
				{Size: 0.012, Price: 3346.709952711325},
				{Size: 0.015, Price: 3346.700261990111},
				{Size: 0.012, Price: 3346.680880547683},
				{Size: 0.021, Price: 3346.6711898264693},
				{Size: 0.033, Price: 3346.6421176628264},
				{Size: 0.027, Price: 3346.6130454991844},
				{Size: 0.59, Price: 3346.4124246485567},
				{Size: 2.9644734252274776, Price: 3343.4441414468833},
			},
			expectedOneToZeroPriceLevels: []PriceLevel{
				// {Size:0.15671334804153958, Price:0.00029728778191376684}}
				{Size: 0.1567133481576093, Price: 0.00029728778191376684},
			},
		},
		{
			name:        "amountIn0 higher than total level size",
			amountIn0:   bignumber.NewBig("5000000000000000000"),
			expectedErr: ErrAmountInIsGreaterThanTotalLevelSize,
		},
		{
			name:        "amountIn1 higher than total level size",
			amountIn1:   bignumber.NewBig("300000000000000000000000"),
			expectedErr: ErrAmountInIsGreaterThanTotalLevelSize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPoolSimulator(entityPoolData)
			assert.NoError(t, err)
			token, amountIn := entityPoolData.Tokens[0].Address, tt.amountIn0
			if amountIn == nil {
				token, amountIn = entityPoolData.Tokens[1].Address, tt.amountIn1
			}
			limit := swaplimit.NewInventory("mx-trading", p.CalculateLimit())

			assert.Equal(t, entityPoolData.Reserves[0], limit.GetLimit(p.token0.Address).String())
			assert.Equal(t, entityPoolData.Reserves[1], limit.GetLimit(p.token1.Address).String())

			calcAmountOutResult, err := p.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: token, Amount: amountIn},
				Limit:         limit,
			})

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
				return
			}

			assert.NoError(t, err)
			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: token, Amount: amountIn},
				TokenAmountOut: *calcAmountOutResult.TokenAmountOut,
				SwapLimit:      limit,
			})

			assert.Equal(t, tt.expectedZeroToOnePriceLevels, p.ZeroToOnePriceLevels)
			assert.Equal(t, tt.expectedOneToZeroPriceLevels, p.OneToZeroPriceLevels)

			tokenInIndex := p.GetTokenIndex(token)
			assert.Equal(t,
				new(big.Int).Add(p.GetReserves()[tokenInIndex], amountIn).String(),
				limit.GetLimit(token).String(),
			)
			assert.Equal(t,
				new(big.Int).Sub(p.GetReserves()[1-tokenInIndex], calcAmountOutResult.TokenAmountOut.Amount).String(),
				limit.GetLimit(calcAmountOutResult.TokenAmountOut.Token).String(),
			)
		})
	}
}
