package angletransmuter

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Tx on chain: 0xd1462167e4f79bdd69dcccdc9ff9c0b6fed665b2e44de993ebd8285fb0079411
func getMintPoolUSD() PoolSimulator {
	return PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Tokens: []string{"0xbeef01735c132ada46aa9aa4c54623caa92a64cb", "0x0000206329b97db379d5e1bf586bbdb969c63274"},
		}},
		Tokens: []*entity.PoolToken{
			{Address: "0xbeef01735c132ada46aa9aa4c54623caa92a64cb", Decimals: 18},
			{Address: "0x0000206329b97db379d5e1bf586bbdb969c63274", Decimals: 18},
		},
		Transmuter: TransmuterState{
			TotalStablecoinIssued: setUInt("12394643135438408381545155"),
			Collaterals: map[string]CollateralState{
				"0xbeef01735c132ada46aa9aa4c54623caa92a64cb": {
					StablecoinsIssued: setUInt("11160955122463689430059999"),
					Fees: Fees{
						XFeeMint: []*uint256.Int{
							uint256.NewInt(0), uint256.NewInt(940000000), uint256.NewInt(950000000),
						},
						YFeeMint: []*uint256.Int{
							uint256.NewInt(500000), uint256.NewInt(500000), uint256.NewInt(999999999999),
						},
						XFeeBurn: []*uint256.Int{uint256.NewInt(1000000000), uint256.NewInt(310000000), uint256.NewInt(300000000)},
						YFeeBurn: []*uint256.Int{uint256.NewInt(500000), uint256.NewInt(500000), uint256.NewInt(999000000)},
					},
					Config: Oracle{
						TargetType: MAX,
						OracleType: MORPHO_ORACLE,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(0),
							BurnRatioDeviation: uint256.NewInt(500000000000000),
						},
						TargetFeed: OracleFeed{
							Max: setUInt("1089431838480000000"),
						},
						OracleFeed: OracleFeed{
							IsPyth:      false,
							IsChainLink: false,
							IsMorpho:    true,
							Morpho: Morpho{
								Active:              true,
								NormalizationFactor: setUInt("1000000000000000000"),
								Price:               setUInt("1089563197304690000000000000000000000"),
							},
						},
					},
				},
			},
		},
	}
}

func TestCollat1_ReadMint_USD(t *testing.T) {
	// 0xd1462167e4f79bdd69dcccdc9ff9c0b6fed665b2e44de993ebd8285fb0079411?trace=0.5.0.0.2.0.5.3.1.1.14.1
	p := getMintPoolUSD()
	targetPrice, err := p._read(MAX, p.Transmuter.Collaterals["0xbeef01735c132ada46aa9aa4c54623caa92a64cb"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1089431838480000000"), targetPrice)

	oraclePrice, err := p._read(MORPHO_ORACLE, p.Transmuter.Collaterals["0xbeef01735c132ada46aa9aa4c54623caa92a64cb"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1089563197304690000"), oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0xbeef01735c132ada46aa9aa4c54623caa92a64cb")
	assert.Nil(t, err)
	assert.Equal(t, targetPrice, target)
	assert.Equal(t, oraclePrice, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, err := p._readMint("0xbeef01735c132ada46aa9aa4c54623caa92a64cb")
	assert.Nil(t, err)
	assert.Equal(t, targetPrice, oracleValue)
}

func Test_quoteMintExactInput_USD(t *testing.T) {
	p := getMintPoolUSD()
	amountIn := setUInt("1198444191209609591202505")
	oracleValue, err := p._readMint("0xbeef01735c132ada46aa9aa4c54623caa92a64cb")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1089431838480000000"), oracleValue)
	amountOut, err := _quoteMintExactInput(
		oracleValue,
		amountIn,
		p.Transmuter.Collaterals["0xbeef01735c132ada46aa9aa4c54623caa92a64cb"].Fees,
		p.Transmuter.Collaterals["0xbeef01735c132ada46aa9aa4c54623caa92a64cb"].StablecoinsIssued,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0xbeef01735c132ada46aa9aa4c54623caa92a64cb"].StablecoinsIssued),
		nil, 18,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1304970773158582340816377"), amountOut)
}
