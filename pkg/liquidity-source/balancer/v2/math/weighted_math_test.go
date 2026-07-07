package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestCalcOutGivenIn(t *testing.T) {
	t.Parallel()
	t.Run("1.should return OK", func(t *testing.T) {
		// input
		balanceIn := uint256.MustFromDecimal("2133741937219414819371293")
		weightIn := uint256.MustFromDecimal("10")
		balanceOut := uint256.MustFromDecimal("548471973423647283412313")
		weightOut := uint256.MustFromDecimal("20")
		amountIn := uint256.MustFromDecimal("21481937129313123729")

		// expected
		expected := "2760912942840907991"

		// calculation
		result, err := WeightedMath.CalcOutGivenIn(balanceIn, weightIn, balanceOut, weightOut, amountIn)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("2.should return OK", func(t *testing.T) {
		// input
		balanceIn := uint256.MustFromDecimal("92174932794319461529478329")
		weightIn := uint256.MustFromDecimal("15")
		balanceOut := uint256.MustFromDecimal("2914754379179427149231562")
		weightOut := uint256.MustFromDecimal("5")
		amountIn := uint256.MustFromDecimal("14957430248210")

		// expected
		expected := "1389798609308"

		// calculation
		result, err := WeightedMath.CalcOutGivenIn(balanceIn, weightIn, balanceOut, weightOut, amountIn)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("3.should return OK", func(t *testing.T) {
		// input
		balanceIn := uint256.MustFromDecimal("28430120665864259131432")
		weightIn := uint256.MustFromDecimal("100000000000000000")
		balanceOut := uint256.MustFromDecimal("10098902157921113397")
		weightOut := uint256.MustFromDecimal("30000000000000000")
		amountIn := uint256.MustFromDecimal("6125185803357185587126")

		// expected
		expected := "4828780052665314529"

		// calculation
		result, err := WeightedMath.CalcOutGivenIn(balanceIn, weightIn, balanceOut, weightOut, amountIn)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("4.should return error exceed amount in ratio", func(t *testing.T) {
		// input
		balanceIn := uint256.MustFromDecimal("92174932794319461529478329")
		weightIn := uint256.MustFromDecimal("15")
		balanceOut := uint256.MustFromDecimal("2914754379179427149231562")
		weightOut := uint256.MustFromDecimal("5")
		amountIn := uint256.MustFromDecimal("92174932794319461529478329")

		// calculation
		_, err := WeightedMath.CalcOutGivenIn(balanceIn, weightIn, balanceOut, weightOut, amountIn)

		// assert
		assert.ErrorIs(t, err, ErrMaxInRatio)
	})
}

func Test_weightedMath_CalcInGivenOut(t *testing.T) {
	t.Parallel()
	type args struct {
		balanceIn  *uint256.Int
		weightIn   *uint256.Int
		balanceOut *uint256.Int
		weightOut  *uint256.Int
		amountOut  *uint256.Int
	}
	tests := []struct {
		name    string
		args    args
		want    *uint256.Int
		wantErr error
	}{
		{
			name: "1. should return OK",
			args: args{
				balanceIn:  uint256.MustFromDecimal("2133741937219414819371293"),
				weightIn:   uint256.MustFromDecimal("10"),
				balanceOut: uint256.MustFromDecimal("548471973423647283412313"),
				weightOut:  uint256.MustFromDecimal("20"),
				amountOut:  uint256.MustFromDecimal("21481937129313123729"),
			},
			want:    uint256.MustFromDecimal("167153858139050441751"),
			wantErr: nil,
		},
		{
			name: "2. should return OK",
			args: args{
				balanceIn:  uint256.MustFromDecimal("92174932794319461529478329"),
				weightIn:   uint256.MustFromDecimal("15"),
				balanceOut: uint256.MustFromDecimal("2914754379179427149231562"),
				weightOut:  uint256.MustFromDecimal("5"),
				amountOut:  uint256.MustFromDecimal("14957430248210"),
			},
			want:    uint256.MustFromDecimal("158591027569670"),
			wantErr: nil,
		},
		{
			name: "3. should return OK",
			args: args{
				balanceIn:  uint256.MustFromDecimal("28430120665864259131432"),
				weightIn:   uint256.MustFromDecimal("100000000000000000"),
				balanceOut: uint256.MustFromDecimal("100989021579211133970000"),
				weightOut:  uint256.MustFromDecimal("30000000000000000"),
				amountOut:  uint256.MustFromDecimal("6125185803357185587126"),
			},
			want:    uint256.MustFromDecimal("538695515311227973058"),
			wantErr: nil,
		},
		{
			name: "4. should return error exceed amount out ratio",
			args: args{
				balanceIn:  uint256.MustFromDecimal("92174932794319461529478329"),
				weightIn:   uint256.MustFromDecimal("15"),
				balanceOut: uint256.MustFromDecimal("2914754379179427149231562"),
				weightOut:  uint256.MustFromDecimal("5"),
				amountOut:  uint256.MustFromDecimal("2914754379179427149231562"),
			},
			want:    nil,
			wantErr: ErrMaxOutRatio,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &weightedMath{}
			got, err := l.CalcInGivenOut(tt.args.balanceIn, tt.args.weightIn, tt.args.balanceOut, tt.args.weightOut, tt.args.amountOut)
			if err != nil {
				assert.ErrorIsf(t, err, tt.wantErr, "weightedMath.CalcInGivenOut() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equalf(t, tt.want, got, "weightedMath.CalcInGivenOut() = %v, want %v", got, tt.want)
		})
	}
}
