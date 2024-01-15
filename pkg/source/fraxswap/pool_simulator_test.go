package fraxswap

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	type args struct {
		tokenAmountIn pool.TokenAmount
		tokenOut      string
	}

	pooldata := struct {
		tokens  []string
		reserve []string
		fee     string
	}{[]string{"a", "b"}, []string{"20", "20"}, "9997"}

	testcases := []struct {
		name    string
		args    args
		want    *pool.CalcAmountOutResult
		wantErr error
	}{
		{
			name: "it should yield correct amount",
			args: args{
				tokenAmountIn: pool.TokenAmount{Token: "a", Amount: big.NewInt(20)},
				tokenOut:      "b",
			},
			want: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{Token: "b", Amount: big.NewInt(9)},
				Fee:            &pool.TokenAmount{Token: "a", Amount: big.NewInt(1), AmountUsd: 0},
				Gas:            113276,
			},
			wantErr: nil,
		},
		{
			name: "it should error with invalid amount in",
			args: args{
				tokenAmountIn: pool.TokenAmount{Token: "a", Amount: big.NewInt(0)},
				tokenOut:      "b",
			},
			want:    nil,
			wantErr: ErrInsufficientInputAmount,
		},
		{
			name: "it should not consume all reserve even with large amount",
			args: args{
				tokenAmountIn: pool.TokenAmount{Token: "a", Amount: big.NewInt(9999999999)},
				tokenOut:      "b",
			},
			want: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{Token: "b", Amount: big.NewInt(19)},
				Fee:            &pool.TokenAmount{Token: "a", Amount: big.NewInt(3000000), AmountUsd: 0},
				Gas:            113276,
			},
			wantErr: nil,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPoolSimulator(entity.Pool{
				Exchange: "",
				Type:     "",
				Reserves: pooldata.reserve,
				Tokens:   lo.Map(pooldata.tokens, func(adr string, _ int) *entity.PoolToken { return &entity.PoolToken{Address: adr} }),
				Extra:    fmt.Sprintf("{\"reserve0\": %v, \"reserve1\": %v, \"fee\": %v}", pooldata.reserve[0], pooldata.reserve[1], pooldata.fee),
			})
			require.Nil(t, err)

			got, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(
					pool.CalcAmountOutParams{
						TokenAmountIn: tt.args.tokenAmountIn,
						TokenOut:      tt.args.tokenOut,
						Limit:         nil,
					})
			})

			require.ErrorIs(t, err, tt.wantErr)
			if err == nil {
				assert.Equal(t, tt.want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
				assert.Equal(t, tt.want.TokenAmountOut.Token, got.TokenAmountOut.Token)
				assert.Equal(t, tt.want.TokenAmountOut.AmountUsd, got.TokenAmountOut.AmountUsd)
				assert.Equal(t, tt.want.Fee.Amount, got.Fee.Amount)
				assert.Equal(t, tt.want.Gas, got.Gas)
			}
		})
	}
}
