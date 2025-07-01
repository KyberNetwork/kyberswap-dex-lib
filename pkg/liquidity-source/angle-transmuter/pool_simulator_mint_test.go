package angletransmuter

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Tx on chain: 0x0138aa67f964465cdfc6dcac3581471c63ac044f7dce3d283e75ce23790c7093
func getMintPool() PoolSimulator {
	return PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Tokens: []string{"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8"},
		}},
		Tokens: []*entity.PoolToken{
			{Address: "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", Decimals: 6},
			{Address: "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", Decimals: 18},
		},
		Transmuter: TransmuterState{
			TotalStablecoinIssued: setUInt("11600921906778307242249332"),
			Collaterals: map[string]CollateralState{
				"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c": {
					StablecoinsIssued: setUInt("2404480312662610902608440"),
					Fees: Fees{
						XFeeMint: []*uint256.Int{
							uint256.NewInt(0), uint256.NewInt(690000000), uint256.NewInt(700000000),
						},
						YFeeMint: []*uint256.Int{
							uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(999999999999),
						},
						XFeeBurn: []*uint256.Int{uint256.NewInt(1000000000)},
						YFeeBurn: []*uint256.Int{uint256.NewInt(0)},
					},
					Config: Oracle{
						TargetType: STABLE,
						OracleType: PYTH,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(1000000000000000),
							BurnRatioDeviation: uint256.NewInt(0),
						},
						TargetFeed: OracleFeed{},
						OracleFeed: OracleFeed{
							IsPyth:      true,
							IsChainLink: false,
							IsMorpho:    false,
							Pyth: Pyth{
								Active:       true,
								FeedIds:      []string{"0x76fa85158bf14ede77087fe3ae472f66213f6ea2f5b411cb2de472794990fa5c", "0xa995d00bb36a63cef7fd2c287dc105fc8f3d93779f062f09551b0af3e81ec30b"},
								IsMultiplied: []uint8{1, 0},
								PythState: []PythState{
									{
										Price: uint256.NewInt(115186038),
										Expo:  uint256.MustFromBig(big.NewInt(-8)),
									},
									{
										Price: uint256.NewInt(115218),
										Expo:  uint256.MustFromBig(big.NewInt(-5)),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

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
