package angletransmuter

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

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
					IsManaged:                 false,
					IsBurnLive:                true,
					IsMintLive:                true,
					StablecoinsIssued:         setUInt("2404480312662610902608440"),
					StablecoinsFromCollateral: setUInt("11056546207338107089243622"),
					Balance:                   setUInt("10000000000000000000000"),
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
							Pyth: &Pyth{
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
					IsManaged:                 false,
					IsBurnLive:                true,
					IsMintLive:                true,
					StablecoinsIssued:         setUInt("2404480312662610902608440"),
					StablecoinsFromCollateral: setUInt("11056546207338107089243622"),
					Balance:                   setUInt("10000000000000000000000"),
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
							Chainlink: &Chainlink{
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
					IsManaged:                 false,
					IsBurnLive:                true,
					IsMintLive:                true,
					StablecoinsIssued:         setUInt("2404480312662610902608440"),
					StablecoinsFromCollateral: setUInt("11056546207338107089243622"),
					Balance:                   setUInt("10000000000000000000000"),
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
							Chainlink: &Chainlink{
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
					IsManaged:                 false,
					IsBurnLive:                true,
					IsMintLive:                true,
					StablecoinsIssued:         setUInt("2404480312662610902608440"),
					StablecoinsFromCollateral: setUInt("11056546207338107089243622"),
					Balance:                   setUInt("10000000000000000000000"),
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
							Morpho: &Morpho{
								NormalizationFactor: setUInt("1000000000000000000"),
								Price:               setUInt("1030046000000000000000000000000000000"),
							},
						},
						OracleFeed: OracleFeed{},
					},
				},
				"0x5f7827fdeb7c20b443265fc2f40845b715385ff2": {
					// 0x4d103fff4e73fc78533cde4aa4fe2cce1da044b4fc4d9439d4f0fd997b2f1e02?trace=0.0.1.1.0.1.1.1.25.84
					IsManaged:                 false,
					IsBurnLive:                true,
					IsMintLive:                true,
					StablecoinsIssued:         setUInt("2404480312662610902608440"),
					StablecoinsFromCollateral: setUInt("11056546207338107089243622"),
					Balance:                   setUInt("10000000000000000000000"),
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

// Tx on chain: 0x0138aa67f964465cdfc6dcac3581471c63ac044f7dce3d283e75ce23790c7093
func getMintPool() PoolSimulator {
	return PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Tokens: []string{"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8"},
		}},
		Decimals: []uint8{6, 18},
		Transmuter: TransmuterState{
			TotalStablecoinIssued: setUInt("11600921906778307242249332"),
			Collaterals: map[string]CollateralState{
				"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c": {
					IsManaged:                 false,
					IsBurnLive:                true,
					IsMintLive:                true,
					StablecoinsIssued:         setUInt("2404480312662610902608440"),
					StablecoinsFromCollateral: setUInt("11056546207338107089243622"),
					Balance:                   setUInt("10000000000000000000000"),
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
							Pyth: &Pyth{
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

// Tx on chain: 0xd1462167e4f79bdd69dcccdc9ff9c0b6fed665b2e44de993ebd8285fb0079411
func getMintPoolUSD() PoolSimulator {
	return PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Tokens: []string{"0xbeef01735c132ada46aa9aa4c54623caa92a64cb", "0x0000206329b97db379d5e1bf586bbdb969c63274"},
		}},
		Decimals: []uint8{18, 18},
		Transmuter: TransmuterState{
			TotalStablecoinIssued: setUInt("12394643135438408381545155"),
			Collaterals: map[string]CollateralState{
				"0xbeef01735c132ada46aa9aa4c54623caa92a64cb": {
					IsManaged:         false,
					IsBurnLive:        true,
					IsMintLive:        true,
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
							Morpho: &Morpho{
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

func getParallelPool() PoolSimulator {
	return PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Tokens: []string{
				"0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE", // scUSD
				"0xA19ebd8f9114519bF947671021c01d152c3777E4", // ygami_scUSD
				"0x08417cdb7F52a5021bB4eb6E0deAf3f295c3f182", // USDp
			},
		}},
		Decimals: []uint8{6, 6, 18},
		Transmuter: TransmuterState{
			TotalStablecoinIssued: setUInt("493987443536936298347938"),
			Collaterals: map[string]CollateralState{
				"0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE": {
					IsManaged:                 false,
					IsBurnLive:                true,
					IsMintLive:                true,
					StablecoinsIssued:         setUInt("6359003924739830000"),
					StablecoinsFromCollateral: setUInt("11056546207338107089243622"),
					Balance:                   setUInt("10000000000000000000000"),
					Fees: Fees{
						XFeeMint: []*uint256.Int{uint256.NewInt(0)},
						YFeeMint: []*uint256.Int{uint256.NewInt(0)},
						XFeeBurn: []*uint256.Int{uint256.NewInt(1000000000)},
						YFeeBurn: []*uint256.Int{uint256.NewInt(0)},
					},
					Config: Oracle{
						TargetType: STABLE,
						OracleType: CHAINLINK_FEEDS,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(500000000000000),
							BurnRatioDeviation: uint256.NewInt(0),
						},
						TargetFeed: OracleFeed{},
						OracleFeed: OracleFeed{
							IsChainLink: true,
							IsMorpho:    false,
							Chainlink: &Chainlink{
								CircuitChainlink: []common.Address{
									common.HexToAddress("0xACE5e348a341a740004304c2c228Af1A4581920F"),
								},
								CircuitChainIsMultiplied: []uint8{1},
								Answers: []*uint256.Int{
									setUInt("99860115"),
								},
								ChainlinkDecimals: []uint8{8},
							},
						},
					},
				},
				"0xA19ebd8f9114519bF947671021c01d152c3777E4": {
					IsManaged:                 false,
					IsBurnLive:                true,
					IsMintLive:                true,
					StablecoinsIssued:         setUInt("975625588650474856"),
					StablecoinsFromCollateral: setUInt("11056546207338107089243622"),
					Balance:                   setUInt("10000000000000000000000"),
					Fees: Fees{
						XFeeMint: []*uint256.Int{uint256.NewInt(0)},
						YFeeMint: []*uint256.Int{uint256.NewInt(0)},
						XFeeBurn: []*uint256.Int{uint256.NewInt(1000000000)},
						YFeeBurn: []*uint256.Int{uint256.NewInt(500000)},
					},
					Config: Oracle{
						TargetType: MAX,
						OracleType: MORPHO_ORACLE,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(0),
							BurnRatioDeviation: uint256.NewInt(500000000000000),
						},
						TargetFeed: OracleFeed{
							Max: setUInt("998767916392050000"),
						},
						OracleFeed: OracleFeed{
							IsChainLink: false,
							IsMorpho:    true,
							Morpho: &Morpho{
								NormalizationFactor: setUInt("1000000000000000000"),
								Price:               setUInt("998775905201250000000000000000000000"),
							},
						},
					},
				},
			},
		},
	}
}
