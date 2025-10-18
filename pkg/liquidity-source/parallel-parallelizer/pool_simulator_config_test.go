package parallelparallelizer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

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
			Tokens: []string{
				"0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE", // scUSD
				"0xA19ebd8f9114519bF947671021c01d152c3777E4", // ygami_scUSD
				"0x08417cdb7F52a5021bB4eb6E0deAf3f295c3f182", // USDp
			},
		}},
		Decimals: []uint8{6, 6, 18},
		Parallelizer: ParallelizerState{
			TotalStablecoinIssued: setUInt("493987443536936298347938"),
			Collaterals: map[string]CollateralState{
				"0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE": {
					StablecoinsIssued: setUInt("6359003924739830000"),
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
							Chainlink: Chainlink{
								Active:       true,
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
					StablecoinsIssued: setUInt("975625588650474856"),
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
							Morpho: Morpho{
								Active:              true,
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
