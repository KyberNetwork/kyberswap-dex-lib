package angletransmuter

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Tx on chain: 0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02
func setUInt(s string) *uint256.Int {
	bigInt, ok := big.NewInt(0).SetString(s, 10)
	if !ok {
		return nil
	}
	return uint256.MustFromBig(bigInt)
}

func getPool() PoolSimulator {
	return PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Tokens: []string{"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8"},
		}},
		Decimals: []uint8{6, 18},
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
				"0x2f123cf3f37ce3328cc9b5b8415f9ec5109b45e7": {
					//0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1.1.25.32
					Config: Oracle{
						TargetType: MAX,
						OracleType: CHAINLINK_FEEDS,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(0),
							BurnRatioDeviation: uint256.NewInt(1000000000000000),
						},
						TargetFeed: OracleFeed{
							Max: setUInt("124535000000000000000"),
						},
						OracleFeed: OracleFeed{
							IsPyth:      false,
							IsChainLink: true,
							IsMorpho:    false,
							Chainlink: Chainlink{
								Active: true,
								CircuitChainlink: []common.Address{
									common.HexToAddress("0x6E27A25999B3C665E44D903B2139F5a4Be2B6C26"),
								},
								CircuitChainIsMultiplied: []uint8{1},
								Answers: []*uint256.Int{
									setUInt("12429000000"),
								},
								ChainlinkDecimals: []uint8{8},
							},
						},
					},
				},
				"0x3f95aa88ddbb7d9d484aa3d482bf0a80009c52c9": {
					//0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1.1.25.58
					Config: Oracle{
						TargetType: MAX,
						OracleType: CHAINLINK_FEEDS,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(0),
							BurnRatioDeviation: uint256.NewInt(2000000000000000),
						},
						TargetFeed: OracleFeed{
							Max: setUInt("5462000000000000000"),
						},
						OracleFeed: OracleFeed{
							IsPyth:      false,
							IsChainLink: true,
							IsMorpho:    false,
							Chainlink: Chainlink{
								Active: true,
								CircuitChainlink: []common.Address{
									common.HexToAddress("0x475855DAe09af1e3f2d380d766b9E630926ad3CE"),
								},
								CircuitChainIsMultiplied: []uint8{1},
								Answers: []*uint256.Int{
									setUInt("546200000"),
								},
								ChainlinkDecimals: []uint8{8},
							},
						},
					},
				},
				"0x3ee320c9f73a84d1717557af00695a34b26d1f1d": {
					//0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1.1.25.72
					Config: Oracle{
						TargetType: MORPHO_ORACLE,
						OracleType: NO_ORACLE,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(0),
							BurnRatioDeviation: uint256.NewInt(0),
						},
						TargetFeed: OracleFeed{
							IsPyth:      false,
							IsChainLink: false,
							IsMorpho:    true,
							Morpho: Morpho{
								Active:              true,
								NormalizationFactor: setUInt("1000000000000000000"),
								Price:               setUInt("1030046000000000000000000000000000000"),
							},
						},
						OracleFeed: OracleFeed{},
					},
				},
				"0x5f7827fdeb7c20b443265fc2f40845b715385ff2": {
					// 0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1.1.25.84
					Config: Oracle{
						TargetType: STABLE,
						OracleType: NO_ORACLE,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(0),
							BurnRatioDeviation: uint256.NewInt(0),
						},
						TargetFeed: OracleFeed{},
						OracleFeed: OracleFeed{},
					},
				},
			},
		},
	}
}

func TestCollat1_ReadBurn(t *testing.T) {
	// 0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1.1.25.6
	p := getPool()
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
	oracleValue, ratio, err := p._readBurn("0x1abaea1f7c830bd89acc67ec4af516284b1bc33c")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
	assert.Equal(t, BASE_18, ratio)
}

func TestCollat2_ReadBurn(t *testing.T) {
	// 0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1.1.25.32
	p := getPool()
	targetPrice, err := p._read(MAX, p.Transmuter.Collaterals["0x2f123cf3f37ce3328cc9b5b8415f9ec5109b45e7"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("124535000000000000000"), targetPrice)
	oraclePrice, err := p._read(CHAINLINK_FEEDS, p.Transmuter.Collaterals["0x2f123cf3f37ce3328cc9b5b8415f9ec5109b45e7"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("124290000000000000000"), oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0x2f123cf3f37ce3328cc9b5b8415f9ec5109b45e7")
	assert.Nil(t, err)
	assert.Equal(t, targetPrice, target)
	assert.Equal(t, oraclePrice, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, ratio, err := p._readBurn("0x2f123cf3f37ce3328cc9b5b8415f9ec5109b45e7")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
	assert.Equal(t, setUInt("998032681575460713"), ratio)
}

func TestCollat3_ReadBurn(t *testing.T) {
	// 0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1.1.25.58
	p := getPool()
	targetPrice, err := p._read(MAX, p.Transmuter.Collaterals["0x3f95aa88ddbb7d9d484aa3d482bf0a80009c52c9"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("5462000000000000000"), targetPrice)
	oraclePrice, err := p._read(CHAINLINK_FEEDS, p.Transmuter.Collaterals["0x3f95aa88ddbb7d9d484aa3d482bf0a80009c52c9"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("5462000000000000000"), oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0x3f95aa88ddbb7d9d484aa3d482bf0a80009c52c9")
	assert.Nil(t, err)
	assert.Equal(t, targetPrice, target)
	assert.Equal(t, oraclePrice, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, ratio, err := p._readBurn("0x3f95aa88ddbb7d9d484aa3d482bf0a80009c52c9")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
	assert.Equal(t, setUInt("1000000000000000000"), ratio)
}

func TestCollat4_ReadBurn(t *testing.T) {
	p := getPool()
	targetPrice, err := p._read(MORPHO_ORACLE, p.Transmuter.Collaterals["0x3ee320c9f73a84d1717557af00695a34b26d1f1d"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1030046000000000000"), targetPrice)
	oraclePrice, err := p._read(NO_ORACLE, p.Transmuter.Collaterals["0x3ee320c9f73a84d1717557af00695a34b26d1f1d"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1030046000000000000"), oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0x3ee320c9f73a84d1717557af00695a34b26d1f1d")
	assert.Nil(t, err)
	assert.Equal(t, targetPrice, target)
	assert.Equal(t, oraclePrice, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, ratio, err := p._readBurn("0x3ee320c9f73a84d1717557af00695a34b26d1f1d")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
	assert.Equal(t, setUInt("1000000000000000000"), ratio)
}

func TestCollat5_ReadBurn(t *testing.T) {
	p := getPool()
	targetPrice, err := p._read(STABLE, p.Transmuter.Collaterals["0x5f7827fdeb7c20b443265fc2f40845b715385ff2"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1000000000000000000"), targetPrice)
	oraclePrice, err := p._read(NO_ORACLE, p.Transmuter.Collaterals["0x5f7827fdeb7c20b443265fc2f40845b715385ff2"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1000000000000000000"), oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0x5f7827fdeb7c20b443265fc2f40845b715385ff2")
	assert.Nil(t, err)
	assert.Equal(t, targetPrice, target)
	assert.Equal(t, oraclePrice, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, ratio, err := p._readBurn("0x5f7827fdeb7c20b443265fc2f40845b715385ff2")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
	assert.Equal(t, setUInt("1000000000000000000"), ratio)
}

func TestGetBurnOracle(t *testing.T) {
	p := getPool()
	oracleValue, minRatio, err := p._getBurnOracle("0x1abaea1f7c830bd89acc67ec4af516284b1bc33c")
	assert.Nil(t, err)
	assert.Equal(t, BASE_18, oracleValue)
	assert.Equal(t, uint256.NewInt(998032681575460713), minRatio)
}

func Test_quoteBurnExactInput(t *testing.T) {
	// 0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1
	p := getPool()
	amountIn := setUInt("3390079323519859415728")
	oracleValue, minRatio, err := p._getBurnOracle("0x1abaea1f7c830bd89acc67ec4af516284b1bc33c")
	assert.Nil(t, err)
	assert.Equal(t, BASE_18, oracleValue)
	assert.Equal(t, uint256.NewInt(998032681575460713), minRatio)
	amountOutAfterFee, err := _quoteFees(
		p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].Fees,
		BurnExactInput,
		amountIn,
		p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].StablecoinsIssued,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].StablecoinsIssued),
	)
	assert.Nil(t, err)
	assert.Equal(t, amountIn, amountOutAfterFee)

	amountOut, err := _quoteBurnExactInput(
		oracleValue, minRatio, amountIn,
		p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].Fees,
		p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].StablecoinsIssued,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0x1abaea1f7c830bd89acc67ec4af516284b1bc33c"].StablecoinsIssued),
		6,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("3383409958"), amountOut)
}

func TestCalcAmountOut(t *testing.T) {
	p := getPool()
	res, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8",
				Amount: setUInt("3390079323519859415728").ToBig(),
			},
			TokenOut: "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c",
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("3383409958").ToBig(), res.TokenAmountOut.Amount)
}
