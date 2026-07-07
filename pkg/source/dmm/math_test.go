package dmm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAmountOut(t *testing.T) {
	t.Parallel()
	type args struct {
		amountIn       *big.Int
		reserveIn      *big.Int
		reserveOut     *big.Int
		vReserveIn     *big.Int
		vReserveOut    *big.Int
		feeInPrecision *big.Int
	}
	tests := []struct {
		name    string
		args    args
		want    *big.Int
		wantErr error
	}{
		{
			name: "it should return correct amount out",
			args: args{
				amountIn:       NewBig10("1000000"),
				reserveIn:      NewBig10("76640419139"),
				reserveOut:     NewBig10("74588249503"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202296641991668"),
			},
			want:    NewBig10("999659"),
			wantErr: nil,
		},
		{
			name: "it should return correct amount out",
			args: args{
				amountIn:       NewBig10("10000000"),
				reserveIn:      NewBig10("76640419139"),
				reserveOut:     NewBig10("74588249503"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202140782934098"),
			},
			want:    NewBig10("9996598"),
			wantErr: nil,
		},
		{
			name: "it should return correct amount out",
			args: args{
				amountIn:       NewBig10("100000000"),
				reserveIn:      NewBig10("76640419139"),
				reserveOut:     NewBig10("74588249503"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202106270324662"),
			},
			want:    NewBig10("99965391"),
			wantErr: nil,
		},
		{
			name: "it should return error ErrInsufficientInputAmount when amountIn < 0",
			args: args{
				amountIn:       NewBig10("-1"),
				reserveIn:      NewBig10("76640419139"),
				reserveOut:     NewBig10("74588249503"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202106270324662"),
			},
			want:    nil,
			wantErr: ErrInsufficientInputAmount,
		},
		{
			name: "it should return error ErrInsufficientLiquidity when reserveIn = 0",
			args: args{
				amountIn:       NewBig10("1000000"),
				reserveIn:      NewBig10("0"),
				reserveOut:     NewBig10("74588249503"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202106270324662"),
			},
			want:    nil,
			wantErr: ErrInsufficientLiquidity,
		},
		{
			name: "it should return error ErrInsufficientLiquidity when reserveIn < 0",
			args: args{
				amountIn:       NewBig10("1000000"),
				reserveIn:      NewBig10("-76640419139"),
				reserveOut:     NewBig10("74588249503"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202106270324662"),
			},
			want:    nil,
			wantErr: ErrInsufficientLiquidity,
		},
		{
			name: "it should return error ErrInsufficientLiquidity when reserveOut = 0",
			args: args{
				amountIn:       NewBig10("1000000"),
				reserveIn:      NewBig10("76640419139"),
				reserveOut:     NewBig10("0"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202106270324662"),
			},
			want:    nil,
			wantErr: ErrInsufficientLiquidity,
		},
		{
			name: "it should return error ErrInsufficientLiquidity when reserveOut < 0",
			args: args{
				amountIn:       NewBig10("1000000"),
				reserveIn:      NewBig10("76640419139"),
				reserveOut:     NewBig10("-74588249503"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202106270324662"),
			},
			want:    nil,
			wantErr: ErrInsufficientLiquidity,
		},
		{
			name: "it should return error ErrInsufficientLiquidity when amountOut >= reserveOut",
			args: args{
				amountIn:       NewBig10("100000000000000000"),
				reserveIn:      NewBig10("76640419139"),
				reserveOut:     NewBig10("-74588249503"),
				vReserveIn:     NewBig10("14944505875836"),
				vReserveOut:    NewBig10("14942453706200"),
				feeInPrecision: NewBig10("202106270324662"),
			},
			want:    nil,
			wantErr: ErrInsufficientLiquidity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAmountOut(
				tt.args.amountIn,
				tt.args.reserveIn,
				tt.args.reserveOut,
				tt.args.vReserveIn,
				tt.args.vReserveOut,
				tt.args.feeInPrecision,
			)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.True(t, tt.want.Cmp(got) == 0)
		})
	}
}
