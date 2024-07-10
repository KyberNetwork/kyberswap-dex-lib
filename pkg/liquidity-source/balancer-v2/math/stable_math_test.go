package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestCalculateInvariantV1(t *testing.T) {
	t.Run("1. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(5000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("9999991000000000000000"),
			uint256.MustFromDecimal("99999910000000000056"),
			uint256.MustFromDecimal("8897791020011100123456"),
			uint256.MustFromDecimal("13288977911102200123456"),
			uint256.MustFromDecimal("199791011102200123456"),
			uint256.MustFromDecimal("1997200112156340123456"),
		}

		// expected
		expected := "19410511781031881171190"

		// actual
		result, err := StableMath.CalculateInvariantV1(amp, balances, true)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("2. should return error balance didn't converge", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1500)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("1243021482015219478129472423914"),
			uint256.MustFromDecimal("184305801438975139127489247143"),
			uint256.MustFromDecimal("14830215"),
			uint256.MustFromDecimal("3018454729758945"),
			uint256.MustFromDecimal("3145748925789256143057234"),
			uint256.MustFromDecimal("127312951502507043571956954693255219"),
		}

		// actual
		_, err := StableMath.CalculateInvariantV1(amp, balances, true)
		assert.ErrorIs(t, err, ErrStableGetBalanceDidntConverge)
	})

	t.Run("3. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(10000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("214321579579179247"),
			uint256.MustFromDecimal("42179537589172147219"),
			uint256.MustFromDecimal("1520481514459573495"),
			uint256.MustFromDecimal("414759131324123123"),
			uint256.MustFromDecimal("4219759729147925"),
			uint256.MustFromDecimal("5197345436285624443"),
		}

		// expected
		expected := "10463766246264892100"

		// actual
		result, err := StableMath.CalculateInvariantV1(amp, balances, true)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("4. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(5000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("9999991000000310000"),
			uint256.MustFromDecimal("9999991000031400056"),
			uint256.MustFromDecimal("88973215240111123456"),
			uint256.MustFromDecimal("13288977911102513456"),
			uint256.MustFromDecimal("199791414320012356"),
			uint256.MustFromDecimal("1997200112152140156"),
		}

		// expected
		expected := "63504110862071166478"

		// actual
		result, err := StableMath.CalculateInvariantV1(amp, balances, false)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("5. should return error balance didn't converge", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1500)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("1243021482015219478129472423914"),
			uint256.MustFromDecimal("184305801438975139127489247143"),
			uint256.MustFromDecimal("14830215"),
			uint256.MustFromDecimal("3018454729758945"),
			uint256.MustFromDecimal("3145748925789256143057234"),
			uint256.MustFromDecimal("127312951502507043571956954693255219"),
		}

		// actual
		_, err := StableMath.CalculateInvariantV1(amp, balances, false)
		assert.ErrorIs(t, err, ErrStableGetBalanceDidntConverge)
	})

	t.Run("6. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(10000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("214321579579179247"),
			uint256.MustFromDecimal("42179537589172147219"),
			uint256.MustFromDecimal("1520481514459573495"),
			uint256.MustFromDecimal("414759131324123123"),
			uint256.MustFromDecimal("4219759729147925"),
			uint256.MustFromDecimal("5197345436285624443"),
		}

		// expected
		expected := "10463766246264891996"

		// actual
		result, err := StableMath.CalculateInvariantV1(amp, balances, false)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})
}

func TestCalculateInvariantV2(t *testing.T) {
	t.Run("1. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(5000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("9999991000000000000000"),
			uint256.MustFromDecimal("99999910000000000056"),
			uint256.MustFromDecimal("8897791020011100123456"),
			uint256.MustFromDecimal("13288977911102200123456"),
			uint256.MustFromDecimal("199791011102200123456"),
			uint256.MustFromDecimal("1997200112156340123456"),
		}
		//

		// expected
		expected := "19410511781031881171187"

		// actual
		result, err := StableMath.CalculateInvariantV2(amp, balances)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("2. should return error balance didn't converge", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1500)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("1243021482015219478129472423914"),
			uint256.MustFromDecimal("184305801438975139127489247143"),
			uint256.MustFromDecimal("14830215"),
			uint256.MustFromDecimal("3018454729758945"),
			uint256.MustFromDecimal("3145748925789256143057234"),
			uint256.MustFromDecimal("127312951502507043571956954693255219"),
		}

		// actual
		_, err := StableMath.CalculateInvariantV2(amp, balances)
		assert.ErrorIs(t, err, ErrMulOverflow)
	})

	t.Run("3. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(10000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("214321579579179247"),
			uint256.MustFromDecimal("42179537589172147219"),
			uint256.MustFromDecimal("1520481514459573495"),
			uint256.MustFromDecimal("414759131324123123"),
			uint256.MustFromDecimal("4219759729147925"),
			uint256.MustFromDecimal("5197345436285624443"),
		}

		// expected
		expected := "10463766246264892025"

		// actual
		result, err := StableMath.CalculateInvariantV2(amp, balances)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("4. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(5000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("9999991000000310000"),
			uint256.MustFromDecimal("9999991000031400056"),
			uint256.MustFromDecimal("88973215240111123456"),
			uint256.MustFromDecimal("13288977911102513456"),
			uint256.MustFromDecimal("199791414320012356"),
			uint256.MustFromDecimal("1997200112152140156"),
		}

		// expected
		expected := "63504110862071166482"

		// actual
		result, err := StableMath.CalculateInvariantV2(amp, balances)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("5. should return error balance didn't converge", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1500)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("1243021482015219478129472423914"),
			uint256.MustFromDecimal("184305801438975139127489247143"),
			uint256.MustFromDecimal("14830215"),
			uint256.MustFromDecimal("3018454729758945"),
			uint256.MustFromDecimal("3145748925789256143057234"),
			uint256.MustFromDecimal("127312951502507043571956954693255219"),
		}

		// actual
		_, err := StableMath.CalculateInvariantV2(amp, balances)
		assert.ErrorIs(t, err, ErrMulOverflow)
	})

	t.Run("6. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(10000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("214321579579179247"),
			uint256.MustFromDecimal("42179537589172147219"),
			uint256.MustFromDecimal("1520481514459573495"),
			uint256.MustFromDecimal("414759131324123123"),
			uint256.MustFromDecimal("4219759729147925"),
			uint256.MustFromDecimal("5197345436285624443"),
		}

		// expected
		expected := "10463766246264892025"

		// actual
		result, err := StableMath.CalculateInvariantV2(amp, balances)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})
}

func TestGetTokenBalanceGivenInvariantAndAllOtherBalances(t *testing.T) {
	t.Run("1. should return correct balance", func(t *testing.T) {
		// input
		amp := uint256.NewInt(5000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("9999991000000000000000"),
			uint256.MustFromDecimal("99999910000000000056"),
			uint256.MustFromDecimal("8897791020011100123456"),
			uint256.MustFromDecimal("13288977911102200123456"),
			uint256.MustFromDecimal("199791011102200123456"),
			uint256.MustFromDecimal("1997200112156340123456"),
		}
		invariant := uint256.MustFromDecimal("19410511781031881171190")
		tokenIndex := 2

		// expected
		expected := "8897791020011100123930"

		// actual
		result, err := StableMath.GetTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("2. should return correct balance", func(t *testing.T) {
		// input
		amp := uint256.NewInt(25000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("999999100001354743940000000"),
			uint256.MustFromDecimal("999999100018034329147962946"),
			uint256.MustFromDecimal("889779102000123421312964156"),
			uint256.MustFromDecimal("132889779111022001234531231236"),
			uint256.MustFromDecimal("1997910111022512421400123456"),
			uint256.MustFromDecimal("1997200112151432414246340123456"),
		}
		invariant := uint256.MustFromDecimal("194123410511781031881171190")
		tokenIndex := 5

		// expected
		expected := "45669657055"

		// actual
		result, err := StableMath.GetTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("3. should return correct balance", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("99999001354743940000000"),
			uint256.MustFromDecimal("999999100018029147962946"),
			uint256.MustFromDecimal("889779102000123421312964156"),
			uint256.MustFromDecimal("13977922001234531231236"),
			uint256.MustFromDecimal("1997910111022512421400123456"),
			uint256.MustFromDecimal("199720011414246340123456"),
		}
		invariant := uint256.MustFromDecimal("1941234105117810318810")
		tokenIndex := 3

		// expected
		expected := "1"

		// actual
		result, err := StableMath.GetTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("4. should return error mul overflow", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("99999001354743"),
			uint256.MustFromDecimal("999999109147962946"),
			uint256.MustFromDecimal("88972000123421312964156"),
			uint256.MustFromDecimal("139701234531231236"),
			uint256.MustFromDecimal("199711022512421400123456"),
			uint256.MustFromDecimal("199720011414246340123456"),
		}
		invariant := uint256.MustFromDecimal("1941234102135117810318810")
		tokenIndex := 2

		// actual
		_, err := StableMath.GetTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.ErrorIs(t, err, ErrMulOverflow)
	})

	t.Run("5. should return correct balance", func(t *testing.T) {
		// input
		amp := uint256.NewInt(2222)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("999990011312354743"),
			uint256.MustFromDecimal("999999109147962946"),
			uint256.MustFromDecimal("8897200012342134156"),
			uint256.MustFromDecimal("139701234531231236"),
			uint256.MustFromDecimal("1997110225124214006"),
			uint256.MustFromDecimal("1997200114142123456"),
		}
		invariant := uint256.MustFromDecimal("194123410213511781031")
		tokenIndex := 4

		// expected
		expected := "82106384280816317076136"

		// actual
		result, err := StableMath.GetTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})
}

func Test_stableMath_CalcInGivenOut(t *testing.T) {
	type args struct {
		invariant *uint256.Int
		amp       *uint256.Int
		amountOut *uint256.Int
		balances  []*uint256.Int
		indexIn   int
		indexOut  int
	}

	tests := []struct {
		name    string
		args    args
		want    *uint256.Int
		wantErr error
	}{
		{
			name: "1. should return correct amount in",
			args: args{
				invariant: uint256.MustFromDecimal("19410511781031881171190"),
				amp:       uint256.NewInt(5000),
				amountOut: uint256.MustFromDecimal("1000000000000000000"),
				balances: []*uint256.Int{
					uint256.MustFromDecimal("9999991000000000000000"),
					uint256.MustFromDecimal("99999910000000000056"),
					uint256.MustFromDecimal("8897791020011100123456"),
					uint256.MustFromDecimal("13288977911102200123456"),
					uint256.MustFromDecimal("199791011102200123456"),
					uint256.MustFromDecimal("1997200112156340123456"),
				},
				indexIn:  2,
				indexOut: 5,
			},
			want:    uint256.MustFromDecimal("3207468813445824937"),
			wantErr: nil,
		},
		{
			name: "2. should return correct amount in",
			args: args{
				invariant: uint256.MustFromDecimal("19410511781031881171190"),
				amp:       uint256.NewInt(5000),
				amountOut: uint256.MustFromDecimal("1000000000000000000"),
				balances: []*uint256.Int{
					uint256.MustFromDecimal("9999991000000000000000"),
					uint256.MustFromDecimal("99999910000000000056"),
					uint256.MustFromDecimal("8897791020011100123456"),
					uint256.MustFromDecimal("13288977911102200123456"),
					uint256.MustFromDecimal("199791011102200123456"),
					uint256.MustFromDecimal("1997200112156340123456"),
				},
				indexIn:  5,
				indexOut: 2,
			},
			want:    uint256.MustFromDecimal("311917432254632569"),
			wantErr: nil,
		},
		{
			name: "3. should return correct amount in",
			args: args{
				invariant: uint256.MustFromDecimal("19410511781031881171190"),
				amp:       uint256.NewInt(5000),
				amountOut: uint256.MustFromDecimal("1000000000000000000"),
				balances: []*uint256.Int{
					uint256.MustFromDecimal("9999991000000000000000"),
					uint256.MustFromDecimal("99999910000000000056"),
					uint256.MustFromDecimal("8897791020011100123456"),
					uint256.MustFromDecimal("13288977911102200123456"),
					uint256.MustFromDecimal("199791011102200123456"),
					uint256.MustFromDecimal("1997200112156340123456"),
				},
				indexIn:  2,
				indexOut: 4,
			},
			want:    uint256.MustFromDecimal("28910581650762107883"),
			wantErr: nil,
		},
		{
			name: "4. should return correct amount in",
			args: args{
				invariant: uint256.MustFromDecimal("19410511781031881171190"),
				amp:       uint256.NewInt(5000),
				amountOut: uint256.MustFromDecimal("1000000000000000000"),
				balances: []*uint256.Int{
					uint256.MustFromDecimal("9999991000000000000000"),
					uint256.MustFromDecimal("99999910000000000056"),
					uint256.MustFromDecimal("8897791020011100123456"),
					uint256.MustFromDecimal("13288977911102200123456"),
					uint256.MustFromDecimal("199791011102200123456"),
					uint256.MustFromDecimal("1997200112156340123456"),
				},
				indexIn:  4,
				indexOut: 2,
			},
			want:    uint256.MustFromDecimal("34726458543604573"),
			wantErr: nil,
		},
		{
			name: "5. should return correct amount in",
			args: args{
				invariant: uint256.MustFromDecimal("19410511781031881171190"),
				amp:       uint256.NewInt(5000),
				amountOut: uint256.MustFromDecimal("1000000000000000000"),
				balances: []*uint256.Int{
					uint256.MustFromDecimal("9999991000000000000000"),
					uint256.MustFromDecimal("99999910000000000056"),
					uint256.MustFromDecimal("8897791020011100123456"),
					uint256.MustFromDecimal("13288977911102200123456"),
					uint256.MustFromDecimal("199791011102200123456"),
					uint256.MustFromDecimal("1997200112156340123456"),
				},
				indexIn:  4,
				indexOut: 5,
			},
			want:    uint256.MustFromDecimal("111385489340648494"),
			wantErr: nil,
		},
		{
			name: "6. should return correct amount in",
			args: args{
				invariant: uint256.MustFromDecimal("19410511781031881171190"),
				amp:       uint256.NewInt(5000),
				amountOut: uint256.MustFromDecimal("1000000000000000000"),
				balances: []*uint256.Int{
					uint256.MustFromDecimal("9999991000000000000000"),
					uint256.MustFromDecimal("99999910000000000056"),
					uint256.MustFromDecimal("8897791020011100123456"),
					uint256.MustFromDecimal("13288977911102200123456"),
					uint256.MustFromDecimal("199791011102200123456"),
					uint256.MustFromDecimal("1997200112156340123456"),
				},
				indexIn:  5,
				indexOut: 4,
			},
			want:    uint256.MustFromDecimal("9022853912539345705"),
			wantErr: nil,
		},
		{
			name: "7. should return correct amount in",
			args: args{
				invariant: uint256.MustFromDecimal("19410511781031881171190"),
				amp:       uint256.NewInt(5000),
				amountOut: uint256.MustFromDecimal("1000000000000000000"),
				balances: []*uint256.Int{
					uint256.MustFromDecimal("9999991000000000000000"),
					uint256.MustFromDecimal("99999910000000000056"),
					uint256.MustFromDecimal("8897791020011100123456"),
					uint256.MustFromDecimal("13288977911102200123456"),
					uint256.MustFromDecimal("199791011102200123456"),
					uint256.MustFromDecimal("1997200112156340123456"),
				},
				indexIn:  0,
				indexOut: 1,
			},
			want:    uint256.MustFromDecimal("61979389081735763787"),
			wantErr: nil,
		},
		{
			name: "8. should return correct amount in",
			args: args{
				invariant: uint256.MustFromDecimal("19410511781031881171190"),
				amp:       uint256.NewInt(5000),
				amountOut: uint256.MustFromDecimal("1000000000000000000"),
				balances: []*uint256.Int{
					uint256.MustFromDecimal("9999991000000000000000"),
					uint256.MustFromDecimal("99999910000000000056"),
					uint256.MustFromDecimal("8897791020011100123456"),
					uint256.MustFromDecimal("13288977911102200123456"),
					uint256.MustFromDecimal("199791011102200123456"),
					uint256.MustFromDecimal("1997200112156340123456"),
				},
				indexIn:  1,
				indexOut: 0,
			},
			want:    uint256.MustFromDecimal("16259799285477427"),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &stableMath{}
			got, err := l.CalcInGivenOut(tt.args.invariant, tt.args.amp, tt.args.amountOut, tt.args.balances, tt.args.indexIn, tt.args.indexOut)
			if err != nil {
				assert.ErrorIsf(t, err, tt.wantErr, "stableMath.CalcInGivenOut() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equalf(t, tt.want, got, "stableMath.CalcInGivenOut() = %v, want %v", got, tt.want)
		})
	}
}
