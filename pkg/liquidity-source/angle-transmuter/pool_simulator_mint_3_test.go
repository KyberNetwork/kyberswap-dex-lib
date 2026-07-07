package angletransmuter

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func Test_ReadMint_scUSD(t *testing.T) {
	// txhash sonic: 0x28591d1a220f6998ee9e0015fbf1bb1593bc490d384845f11125a011241c4e55
	p := getParallelPool()
	expectedValue := setUInt("998601150000000000")
	targetPrice, err := p._read(STABLE, p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1000000000000000000"), targetPrice)

	oraclePrice, err := p._read(CHAINLINK_FEEDS, p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE")
	assert.Nil(t, err)
	assert.Equal(t, BASE_18, target)
	assert.Equal(t, expectedValue, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, err := p._readMint("0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
}
func Test_ReadMint_ygami_scUSD(t *testing.T) {
	// txhash sonic: 0x3e38ff0952e64133410ec7f3dc1c34c624118b5fca680048220c50433c3e8580
	p := getParallelPool()
	targetPrice, err := p._read(STABLE, p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1000000000000000000"), targetPrice)

	oraclePrice, err := p._read(MORPHO_ORACLE, p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998775905201250000"), oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0xA19ebd8f9114519bF947671021c01d152c3777E4")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998767916392050000"), target)
	assert.Equal(t, setUInt("998775905201250000"), spot)

	// adjust based on BurnRatioDeviation
	oracleValue, err := p._readMint("0xA19ebd8f9114519bF947671021c01d152c3777E4")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998767916392050000"), oracleValue)
}

func Test_quoteMintExactInput_scUSD(t *testing.T) {
	// txhash sonic: 0x28591d1a220f6998ee9e0015fbf1bb1593bc490d384845f11125a011241c4e55
	p := getParallelPool()
	amountIn := setUInt("8910")
	oracleValue, err := p._readMint("0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998601150000000000"), oracleValue)
	collatInfo := p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"]
	amountOut, err := _quoteMintExactInput(
		oracleValue,
		amountIn,
		&collatInfo,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"].StablecoinsIssued),
		nil, 6,
		p.Transmuter.TotalStablecoinIssued,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("8897536246500000"), amountOut)
}

func Test_quoteMintExactInput_ygami_scUSD(t *testing.T) {
	// txhash sonic: 0x3e38ff0952e64133410ec7f3dc1c34c624118b5fca680048220c50433c3e8580
	p := getParallelPool()
	amountIn := setUInt("5930")
	oracleValue, err := p._readMint("0xA19ebd8f9114519bF947671021c01d152c3777E4")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998767916392050000"), oracleValue)
	collatInfo := p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"]
	amountOut, err := _quoteMintExactInput(
		oracleValue,
		amountIn,
		&collatInfo,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"].StablecoinsIssued),
		nil, 6, p.Transmuter.TotalStablecoinIssued,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("5922693744204856"), amountOut)
}
