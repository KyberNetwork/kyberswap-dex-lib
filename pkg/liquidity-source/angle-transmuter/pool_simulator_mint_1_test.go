package angletransmuter

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestCollat1_ReadMint(t *testing.T) {
	p := getMintPool()
	targetPrice, err := p._read(STABLE, p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1000000000000000000"), targetPrice)

	oraclePrice, err := p._read(PYTH, p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("999722595427797739"), oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0x1abaea1f7c830bd89acc67ec4af516284b1bc33c")
	assert.Nil(t, err)
	assert.Equal(t, BASE_18, target)
	assert.Equal(t, BASE_18, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, err := p._readMint("0x1abaea1f7c830bd89acc67ec4af516284b1bc33c")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
}

func Test_quoteMintExactInput(t *testing.T) {
	p := getMintPool()
	amountIn := setUInt("6783333924")
	oracleValue, err := p._readMint("0x1abaea1f7c830bd89acc67ec4af516284b1bc33c")
	assert.Nil(t, err)
	assert.Equal(t, BASE_18, oracleValue)
	amountOut, err := _quoteMintExactInput(
		oracleValue,
		amountIn,
		p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].Fees,
		p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].StablecoinsIssued,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].StablecoinsIssued),
		nil, 6,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("6783333924000000000000"), amountOut)
}
