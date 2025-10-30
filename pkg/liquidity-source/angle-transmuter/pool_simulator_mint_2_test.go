package angletransmuter

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

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
	collatInfo := p.Transmuter.Collaterals["0xbeef01735c132ada46aa9aa4c54623caa92a64cb"]
	amountOut, err := _quoteMintExactInput(
		oracleValue,
		amountIn,
		&collatInfo,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0xbeef01735c132ada46aa9aa4c54623caa92a64cb"].StablecoinsIssued),
		nil, 18,
		p.Transmuter.TotalStablecoinIssued,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1304970773158582340816377"), amountOut)
}
