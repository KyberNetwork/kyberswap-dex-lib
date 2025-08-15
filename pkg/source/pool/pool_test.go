package pool_test

import (
	"context"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral"
	. "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ctx = context.Background()
)

func TestPool_CanSwapTo(t *testing.T) {
	t.Parallel()
	type fields struct {
		Info PoolInfo
	}
	type args struct {
		address string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			"happy",
			fields{
				Info: PoolInfo{
					Tokens: []string{"token1", "token2"},
				},
			},
			args{
				address: "token1",
			},
			[]string{"token2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				Info: tt.fields.Info,
			}
			assert.Equalf(t, tt.want, p.CanSwapTo(tt.args.address), "CanSwapTo(%v)", tt.args.address)
		})
	}
}

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x609a6465ca0381f9b86993d3173800fd84b87b41","exchange":"gliquid","type":"algebra-integral","timestamp":1755071435,"reserves":["12232465359877777688120","226611325142682450417"],"tokens":[{"address":"0x02c6a2fa58cc01a18b8d9e00ea48d65e4df26c70","symbol":"feUSD","decimals":18,"swappable":true},{"address":"0x5555555555555555555555555555555555555555","symbol":"WHYPE","decimals":18,"swappable":true}],"extra":"{\"liq\":\"32804011787156563075259\",\"gS\":{\"price\":\"11933882232131808749502101104\",\"tick\":-37861,\"lF\":500,\"pC\":193,\"cF\":130,\"un\":true},\"ticks\":[{\"Index\":-38100,\"LiquidityGross\":\"5665794774058427323925\",\"LiquidityNet\":\"5665794774058427323925\"},{\"Index\":-37620,\"LiquidityGross\":\"5665794774058427323925\",\"LiquidityNet\":\"-5665794774058427323925\"}],\"tS\":60,\"tP\":{\"0\":{\"init\":true,\"ts\":1752857520,\"vo\":\"0\",\"tick\":-37844,\"avgT\":-37844},\"19834\":{\"init\":true,\"ts\":1755071433,\"cum\":-82989691247,\"vo\":\"116495786956\",\"tick\":-37855,\"avgT\":-37901,\"wsI\":19179},\"19835\":{\"vo\":\"0\"}},\"vo\":{\"init\":true,\"tpIdx\":19834,\"lastTs\":1755071433},\"dF\":{\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"bF\":2400}}","staticExtra":"{\"poolId\":\"0x609a6465ca0381f9b86993d3173800fd84b87b41\"}","blockNumber":11000154}`), &entityPool)
	poolSim    = lo.Must(integral.NewPoolSimulator(entityPool))
)

func Test_CalcAmountOut(t *testing.T) {
	t.Parallel()
	type args struct {
		poolSim  IPoolSimulator
		idxIn    int
		idxOut   int
		amountIn string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			"happy",
			args{
				poolSim,
				0,
				1,
				"1000000000000000000",
			},
			"22633868018750208",
			require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolSim := tt.args.poolSim
			tokens := poolSim.GetTokens()
			tokenIn := tokens[tt.args.idxIn]
			tokenAmtIn := TokenAmount{
				Token:  tokenIn,
				Amount: bignumber.NewBig10(tt.args.amountIn),
			}
			tokenOut := tokens[tt.args.idxOut]

			res, err := CalcAmountOut(ctx, poolSim, tokenAmtIn, tokenOut, nil)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, res.TokenAmountOut.Amount.String())
		})
	}
}
