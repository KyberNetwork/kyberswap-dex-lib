package etherfiebtc

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var entityPoolStrData = "{\"address\":\"0x6ee3aaccf9f2321e49063c4f8da775ddbd407268\",\"exchange\":\"etherfi-ebtc\"," +
	"\"type\":\"etherfi-ebtc\",\"reserves\":[\"10000000000\",\"10000000000\",\"10000000000\",\"10000000000\"]," +
	"\"tokens\":[{\"address\":\"0x657e8c867d8b37dcc18fa4caead9c45eb088c642\",\"name\":\"ether.fi BTC\",\"symbol\":\"eBTC\",\"decimals\":8,\"swappable\":true}," +
	"{\"address\":\"0x8236a87084f8b84306f72007f36f2618a5634494\",\"name\":\"Lombard Staked Bitcoin\",\"symbol\":\"LBTC\",\"decimals\":8,\"swappable\":true}," +
	"{\"address\":\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\",\"name\":\"Wrapped BTC\",\"symbol\":\"WBTC\",\"decimals\":8,\"swappable\":true}," +
	"{\"address\":\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\",\"name\":\"Coinbase Wrapped BTC\",\"symbol\":\"cbBTC\",\"decimals\":8,\"swappable\":true}]," +
	"\"extra\":\"{\\\"isTellerPaused\\\":false,\\\"shareLockPeriod\\\":0,\\\"assets\\\":{\\\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\\\":{\\\"allowDeposits\\\":true,\\\"allowWithdraws\\\":true,\\\"sharePremium\\\":30},\\\"0x657e8c867d8b37dcc18fa4caead9c45eb088c642\\\":{\\\"allowDeposits\\\":false,\\\"allowWithdraws\\\":false,\\\"sharePremium\\\":0},\\\"0x8236a87084f8b84306f72007f36f2618a5634494\\\":{\\\"allowDeposits\\\":true,\\\"allowWithdraws\\\":true,\\\"sharePremium\\\":0},\\\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\\\":{\\\"allowDeposits\\\":true,\\\"allowWithdraws\\\":true,\\\"sharePremium\\\":0}},\\\"accountantState\\\":{\\\"exchangeRate\\\":100000000,\\\"isPaused\\\":false},\\\"rateProviders\\\":{\\\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\\\":{\\\"isPeggedToBase\\\":false,\\\"rateProvider\\\":\\\"0x0000000000000000000000000000000000000000\\\"},\\\"0x657e8c867d8b37dcc18fa4caead9c45eb088c642\\\":{\\\"isPeggedToBase\\\":false,\\\"rateProvider\\\":\\\"0x0000000000000000000000000000000000000000\\\"},\\\"0x8236a87084f8b84306f72007f36f2618a5634494\\\":{\\\"isPeggedToBase\\\":true,\\\"rateProvider\\\":\\\"0x0000000000000000000000000000000000000000\\\"},\\\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\\\":{\\\"isPeggedToBase\\\":true,\\\"rateProvider\\\":\\\"0x0000000000000000000000000000000000000000\\\"}}}\"," +
	"\"staticExtra\":\"{\\\"accountant\\\":\\\"0x1b293dc39f94157fa0d1d36d7e0090c8b8b8c13f\\\",\\\"base\\\":\\\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\\\",\\\"decimals\\\":8}\"}"

var entityPoolData = entity.Pool{
	Address:  "0x6ee3aaccf9f2321e49063c4f8da775ddbd407268",
	Exchange: DexType,
	Type:     DexType,
	Reserves: entity.PoolReserves{defaultReserves, defaultReserves, defaultReserves, defaultReserves},
	Tokens: []*entity.PoolToken{
		{Address: "0x657e8c867d8b37dcc18fa4caead9c45eb088c642", Decimals: 8, Swappable: true, Name: "ether.fi BTC", Symbol: "eBTC"},
		{Address: "0x8236a87084f8b84306f72007f36f2618a5634494", Decimals: 8, Swappable: true, Name: "Lombard Staked Bitcoin", Symbol: "LBTC"},
		{Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Decimals: 8, Swappable: true, Name: "Wrapped BTC", Symbol: "WBTC"},
		{Address: "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf", Decimals: 8, Swappable: true, Name: "Coinbase Wrapped BTC", Symbol: "cbBTC"},
	},
	Extra:       "{\"isTellerPaused\":false,\"shareLockPeriod\":0,\"assets\":{\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\":{\"allowDeposits\":true,\"allowWithdraws\":true,\"sharePremium\":30},\"0x657e8c867d8b37dcc18fa4caead9c45eb088c642\":{\"allowDeposits\":false,\"allowWithdraws\":false,\"sharePremium\":0},\"0x8236a87084f8b84306f72007f36f2618a5634494\":{\"allowDeposits\":true,\"allowWithdraws\":true,\"sharePremium\":0},\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\":{\"allowDeposits\":true,\"allowWithdraws\":true,\"sharePremium\":0}},\"accountantState\":{\"exchangeRate\":100000000,\"isPaused\":false},\"rateProviders\":{\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\":{\"isPeggedToBase\":false,\"rateProvider\":\"0x0000000000000000000000000000000000000000\"},\"0x657e8c867d8b37dcc18fa4caead9c45eb088c642\":{\"isPeggedToBase\":false,\"rateProvider\":\"0x0000000000000000000000000000000000000000\"},\"0x8236a87084f8b84306f72007f36f2618a5634494\":{\"isPeggedToBase\":true,\"rateProvider\":\"0x0000000000000000000000000000000000000000\"},\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\":{\"isPeggedToBase\":true,\"rateProvider\":\"0x0000000000000000000000000000000000000000\"}}}",
	StaticExtra: "{\"accountant\":\"0x1b293dc39f94157fa0d1d36d7e0090c8b8b8c13f\",\"base\":\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\",\"decimals\":8}",
}

func TestNewPoolSimulator(t *testing.T) {
	entityPoolDataBytes, err := json.Marshal(entityPoolData)
	assert.Equal(t, entityPoolStrData, string(entityPoolDataBytes))

	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{defaultReserves, defaultReserves, defaultReserves, defaultReserves},
		Tokens: []*entity.PoolToken{
			{Address: "0x657e8c867d8b37dcc18fa4caead9c45eb088c642", Decimals: 8, Swappable: true, Name: "ether.fi BTC", Symbol: "eBTC"},
			{Address: "0x8236a87084f8b84306f72007f36f2618a5634494", Decimals: 8, Swappable: true, Name: "Lombard Staked Bitcoin", Symbol: "LBTC"},
			{Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Decimals: 8, Swappable: true, Name: "Wrapped BTC", Symbol: "WBTC"},
			{Address: "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf", Decimals: 8, Swappable: true, Name: "Coinbase Wrapped BTC", Symbol: "cbBTC"},
		},
		Extra:       "{\"isTellerPaused\":false,\"shareLockPeriod\":0,\"assets\":{\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\":{\"allowDeposits\":true,\"allowWithdraws\":true,\"sharePremium\":30},\"0x657e8c867d8b37dcc18fa4caead9c45eb088c642\":{\"allowDeposits\":false,\"allowWithdraws\":false,\"sharePremium\":0},\"0x8236a87084f8b84306f72007f36f2618a5634494\":{\"allowDeposits\":true,\"allowWithdraws\":true,\"sharePremium\":0},\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\":{\"allowDeposits\":true,\"allowWithdraws\":true,\"sharePremium\":0}},\"accountantState\":{\"exchangeRate\":100000000,\"isPaused\":false},\"rateProviders\":{\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\":{\"isPeggedToBase\":false,\"rateProvider\":\"0x0000000000000000000000000000000000000000\"},\"0x657e8c867d8b37dcc18fa4caead9c45eb088c642\":{\"isPeggedToBase\":false,\"rateProvider\":\"0x0000000000000000000000000000000000000000\"},\"0x8236a87084f8b84306f72007f36f2618a5634494\":{\"isPeggedToBase\":true,\"rateProvider\":\"0x0000000000000000000000000000000000000000\"},\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\":{\"isPeggedToBase\":true,\"rateProvider\":\"0x0000000000000000000000000000000000000000\"}}}",
		StaticExtra: "{\"accountant\":\"0x1b293dc39f94157fa0d1d36d7e0090c8b8b8c13f\",\"base\":\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\",\"decimals\":8}",
	})
	assert.NoError(t, err)
	assert.NotNil(t, p)

	assert.Equal(t, []string{"0x8236a87084f8b84306f72007f36f2618a5634494", "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf"}, p.CanSwapTo("0x657e8c867d8b37dcc18fa4caead9c45eb088c642"))
	assert.Equal(t, []string{}, p.CanSwapTo("0x8236a87084f8b84306f72007f36f2618a5634494"))
	assert.Equal(t, []string{}, p.CanSwapTo("0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"))
	assert.Equal(t, []string{}, p.CanSwapTo("0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf"))
}

func TestPoolSimulator_GetAmountOut(t *testing.T) {
	WETH := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	eBTC := "0x657e8c867d8b37dcc18fa4caead9c45eb088c642"
	LBTC := "0x8236a87084f8b84306f72007f36f2618a5634494"
	WBTC := "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"
	cbBTC := "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf"

	tests := []struct {
		name              string
		tokenIn           string
		amountIn          *big.Int
		tokenOut          string
		expectedAmountOut *big.Int
		expectedErr       error
	}{
		{
			name:        "it should return error when tokenIn is not supported",
			tokenIn:     WETH,
			amountIn:    big.NewInt(1_000000000000000000),
			tokenOut:    eBTC,
			expectedErr: ErrTellerAssetNotSupported,
		},
		{
			name:        "it should return error when tokenIn is not supported",
			tokenIn:     eBTC,
			amountIn:    big.NewInt(1_00000000),
			tokenOut:    cbBTC,
			expectedErr: ErrTellerAssetNotSupported,
		},
		{
			name:              "it should return the same amount when tokenIn is 1:1 with the base asset",
			tokenIn:           LBTC,
			amountIn:          big.NewInt(1232134123),
			tokenOut:          eBTC,
			expectedAmountOut: big.NewInt(1232134123),
		},
		{
			name:              "it should return the same amount when tokenIn is 1:1 with the base asset",
			tokenIn:           cbBTC,
			amountIn:          big.NewInt(21321),
			tokenOut:          eBTC,
			expectedAmountOut: big.NewInt(21321),
		},
		{
			name:              "it should return the same amount when tokenIn is 1:1 with the base asset",
			tokenIn:           cbBTC,
			amountIn:          big.NewInt(1),
			tokenOut:          eBTC,
			expectedAmountOut: big.NewInt(1),
		},
		{
			name:              "it should return correct amount with a reduction in share premium (30bps)",
			tokenIn:           WBTC,
			amountIn:          big.NewInt(2928753),
			tokenOut:          eBTC,
			expectedAmountOut: big.NewInt(2919966),
		},
		{
			name:              "it should return correct amount with a reduction in share premium (30bps)",
			tokenIn:           WBTC,
			amountIn:          big.NewInt(1_00000000),
			tokenOut:          eBTC,
			expectedAmountOut: big.NewInt(99700000),
		},
		{
			name:              "",
			tokenIn:           WBTC,
			amountIn:          big.NewInt(1),
			tokenOut:          eBTC,
			expectedAmountOut: bignumber.ZeroBI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := NewPoolSimulator(entityPoolData)

			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tt.tokenIn, Amount: tt.amountIn},
				TokenOut:      tt.tokenOut,
			}

			result, err := p.CalcAmountOut(params)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAmountOut.String(), result.TokenAmountOut.Amount.String())
			}
		})
	}
}
