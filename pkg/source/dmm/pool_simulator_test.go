package dmm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	type fields struct {
		entityPool entity.Pool
	}
	type args struct {
		tokenAmountIn pool.TokenAmount
		tokenOut      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pool.CalcAmountOutResult
		wantErr error
	}{
		{
			name: "it should return result correctly",
			fields: fields{
				entityPool: entity.Pool{
					Exchange:  "kyberswap",
					Type:      "dmm",
					Timestamp: 1685615099,
					Reserves:  entity.PoolReserves{"2766560101102", "1840989218168603319854"},
					Tokens: entity.PoolTokens{
						{
							Address:   "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
							Weight:    50,
							Swappable: true,
						},
						{
							Address:   "0xdd974d5c2e2928dea5f71b9825b8b646686bd200",
							Weight:    50,
							Swappable: true,
						},
					},
					Extra:        "{\"vReserves\":[\"867857435362478004\",\"2348002479022720085946\"],\"feeInPrecision\":\"1503833623506882\"}",
					ReserveUsd:   100000,
					AmplifiedTvl: 100000,
				},
			},
			args: args{
				tokenAmountIn: pool.TokenAmount{
					Token:     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount:    big.NewInt(100000000),
					AmountUsd: 0,
				},
				tokenOut: "0xdd974d5c2e2928dea5f71b9825b8b646686bd200",
			},
			want: &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:     "0xdd974d5c2e2928dea5f71b9825b8b646686bd200",
					Amount:    big.NewInt(270144768388),
					AmountUsd: 0,
				},
				Fee: &pool.TokenAmount{
					Token:     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount:    nil,
					AmountUsd: 0,
				},
				Gas: 65000,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t1 *testing.T) {
			p, err := NewPoolSimulator(tt.fields.entityPool)
			assert.Nil(t1, err)

			got, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(
					pool.CalcAmountOutParams{
						TokenAmountIn: tt.args.tokenAmountIn,
						TokenOut:      tt.args.tokenOut,
						Limit:         nil,
					})
			})

			assert.ErrorIs(t, err, tt.wantErr)
			assert.True(t, tt.want.TokenAmountOut.Amount.Cmp(got.TokenAmountOut.Amount) == 0)
			assert.Equal(t, tt.want.TokenAmountOut.Token, got.TokenAmountOut.Token)
			assert.Equal(t, tt.want.TokenAmountOut.AmountUsd, got.TokenAmountOut.AmountUsd)
			assert.Equal(t, tt.want.Fee.Amount, got.Fee.Amount)
			assert.Equal(t, tt.want.Gas, got.Gas)
		})
	}
}
