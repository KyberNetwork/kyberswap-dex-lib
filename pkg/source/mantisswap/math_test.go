package mantisswap_test

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/mantisswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAmountOut(t *testing.T) {
	from := "0x2791bca1f2de4661ed88a30c99a7a9449aa84174"
	to := "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"
	amount := bignumber.NewBig10("38817001")
	state := &mantisswap.PoolState{
		SwapAllowed: true,
		BaseFee:     bignumber.NewBig10("100"),
		LpRatio:     bignumber.NewBig10("50"),
		SlippageA:   bignumber.NewBig10("8"),
		SlippageN:   bignumber.NewBig10("16"),
		SlippageK:   bignumber.NewBig10("1000000000000000000"),
		LPs: map[string]*mantisswap.LP{
			"0x2791bca1f2de4661ed88a30c99a7a9449aa84174": {
				Address:        "0xe03aec0d08B3158350a9aB99f6Cea7bA9513B889",
				Decimals:       6,
				Asset:          bignumber.NewBig10("3881700128"),
				Liability:      bignumber.NewBig10("3184369687"),
				LiabilityLimit: bignumber.NewBig10("2000000000000"),
			},
			"0xc2132d05d31c914a87c6611c10748aeb04b58e8f": {
				Address:        "0xe8A1eAD2F4c454e319b76fA3325B754C47Ce1820",
				Decimals:       6,
				Asset:          bignumber.NewBig10("1261079342"),
				Liability:      bignumber.NewBig10("1836151875"),
				LiabilityLimit: bignumber.NewBig10("2000000000000"),
			},
			"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063": {
				Address:        "0x4b3BFcaa4F8BD4A276B81C110640dA634723e64B",
				Decimals:       18,
				Asset:          bignumber.NewBig10("1171148472037936744784"),
				Liability:      bignumber.NewBig10("996441749584534394315"),
				LiabilityLimit: bignumber.NewBig10("2000000000000000000000000"),
			},
		},
	}

	toAmount, err := mantisswap.GetAmountOut(from, to, amount, state)

	assert.Nil(t, err)
	assert.Equal(t, "38801087", toAmount.String())
}

func TestGetSlippage(t *testing.T) {
	slippage, err := mantisswap.GetSlippage(
		bignumber.NewBig10("1231175244823262255"),
		&mantisswap.PoolState{
			SlippageA: bignumber.NewBig10("8"),
			SlippageN: bignumber.NewBig10("16"),
			SlippageK: bignumber.NewBig10("1000000000000000000"),
		},
	)

	assert.Nil(t, err)
	assert.Equal(t, "3461443437773", slippage.String())
}
