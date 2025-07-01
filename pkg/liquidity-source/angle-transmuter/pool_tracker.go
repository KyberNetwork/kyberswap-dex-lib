package angletransmuter

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type DecodedOracleConfig struct {
	OracleType      uint8
	TargetType      uint8
	OracleData      []byte
	TargetData      []byte
	Hyperparameters []byte
}

type DecodedHyperparameters struct {
	UserDeviation      *big.Int
	BurnRatioDeviation *big.Int
}

type DecodedIssuedByCollateral struct {
	StablecoinsFromCollateral *big.Int
	StablecoinsIssued         *big.Int
}

type DecodedPyth struct {
	Pyth         common.Address
	FeedIds      [][32]byte
	StalePeriods []uint32
	IsMultiplied []uint8
	QuoteType    uint8
}

type DecodedChainlink struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}

type DecodedPythStateTuple struct {
	DecodedPythState
}
type DecodedPythState struct {
	Price       int64    // price from Pyth
	Conf        uint64   // confidence interval
	Expo        int32    // exponent
	PublishTime *big.Int // publish timestamp
}

type DecodedFeeMints struct {
	XFeeMint []uint64
	YFeeMint []int64
}

type DecodedFeeBurns struct {
	XFeeBurn []uint64
	YFeeBurn []int64
}

type DecodedRedemptionFees struct {
	XRedemptionCurve []uint64
	YRedemptionCurve []int64
}

type DecodedMax struct {
	MaxValue *big.Int
}

type DecodedMorpho struct {
	Oracle              common.Address
	NormalizationFactor *big.Int
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	var collateralList []common.Address
	if _, err := calls.AddCall(&ethrpc.Call{
		ABI:    transmuterABI,
		Target: t.config.Transmuter,
		Method: "getCollateralList",
	}, []any{&collateralList}).Aggregate(); err != nil {
		return p, err
	}

	collateralConfigs := make([]DecodedOracleConfig, len(collateralList))
	feeMints := make([]DecodedFeeMints, len(collateralList))
	feeBurns := make([]DecodedFeeBurns, len(collateralList))
	issuedByCollateral := make([]DecodedIssuedByCollateral, len(collateralList))
	stablecoinCap := make([]*big.Int, len(collateralList))
	isWhitelistedCollateral := make([]bool, len(collateralList))
	collateralWhitelistData := make([][]byte, len(collateralList))
	var totalStablecoinIssued *big.Int
	var redemptionFees DecodedRedemptionFees
	calls = t.ethrpcClient.NewRequest().SetContext(ctx)

	for i, collateral := range collateralList {
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getOracle",
			Params: []any{collateral},
		}, []any{&collateralConfigs[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getCollateralMintFees",
			Params: []any{collateral},
		}, []any{&feeMints[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getCollateralBurnFees",
			Params: []any{collateral},
		}, []any{&feeBurns[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getIssuedByCollateral",
			Params: []any{collateral},
		}, []any{&issuedByCollateral[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "isWhitelistedCollateral",
			Params: []any{collateral},
		}, []any{&isWhitelistedCollateral[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getCollateralWhitelistData",
			Params: []any{collateral},
		}, []any{&collateralWhitelistData[i]})
		if t.config.ChainID != 1 {
			calls.AddCall(&ethrpc.Call{
				ABI:    transmuterABI,
				Target: t.config.Transmuter,
				Method: "getStablecoinCap",
				Params: []any{collateral},
			}, []any{&stablecoinCap[i]})
		}
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    transmuterABI,
		Target: t.config.Transmuter,
		Method: "getRedemptionFees",
		Params: nil,
	}, []any{&redemptionFees})
	calls.AddCall(&ethrpc.Call{
		ABI:    transmuterABI,
		Target: t.config.Transmuter,
		Method: "getTotalIssued",
		Params: nil,
	}, []any{&totalStablecoinIssued})

	if _, err := calls.Aggregate(); err != nil {
		return p, err
	}

	transmuterState := TransmuterState{
		XRedemptionCurve:      redemptionFees.XRedemptionCurve,
		YRedemptionCurve:      redemptionFees.YRedemptionCurve,
		TotalStablecoinIssued: uint256.MustFromBig(totalStablecoinIssued),
		Collaterals:           make(map[string]CollateralState),
	}
	for i := range collateralList {
		if i == 0 {
			continue
		}
		transmuterState.Collaterals[strings.ToLower(collateralList[i].Hex())] = CollateralState{
			Whitelisted:   isWhitelistedCollateral[i],
			WhitelistData: collateralWhitelistData[i],
			Fees: Fees{
				XFeeMint: lo.Map(feeMints[i].XFeeMint, func(item uint64, _ int) *uint256.Int {
					return uint256.NewInt(item)
				}),
				YFeeMint: lo.Map(feeMints[i].YFeeMint, func(item int64, _ int) *uint256.Int {
					return uint256.NewInt(uint64(item))
				}),
				XFeeBurn: lo.Map(feeBurns[i].XFeeBurn, func(item uint64, _ int) *uint256.Int {
					return uint256.NewInt(item)
				}),
				YFeeBurn: lo.Map(feeBurns[i].YFeeBurn, func(item int64, _ int) *uint256.Int {
					return uint256.NewInt(uint64(item))
				}),
			},
			StablecoinsIssued: uint256.MustFromBig(issuedByCollateral[i].StablecoinsFromCollateral),
			StablecoinCap:     uint256.MustFromBig(stablecoinCap[i]),
			Config: Oracle{
				OracleType: OracleReadType(collateralConfigs[i].OracleType),
				TargetType: OracleReadType(collateralConfigs[i].TargetType),
				// ExternalOracle: collateralConfigs[i].ExternalOracle, // TODO:
				OracleFeed: t.getOracleFeed(ctx, OracleReadType(collateralConfigs[i].OracleType), collateralConfigs[i].OracleData),
				TargetFeed: t.getOracleFeed(ctx, OracleReadType(collateralConfigs[i].TargetType), collateralConfigs[i].TargetData),
				Hyperparameters: func() Hyperparameters {
					unpacked, err := HyperparametersArgument.Unpack(collateralConfigs[i].Hyperparameters)
					if err != nil {
						return Hyperparameters{}
					}
					var params DecodedHyperparameters
					if err := HyperparametersArgument.Copy(&params, unpacked); err != nil {
						return Hyperparameters{}
					}
					return Hyperparameters{
						UserDeviation:      uint256.MustFromBig(params.UserDeviation),
						BurnRatioDeviation: uint256.MustFromBig(params.BurnRatioDeviation),
					}
				}(),
			},
		}
	}

	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

func (t *PoolTracker) getOracleFeed(ctx context.Context, oracleType OracleReadType, oracleData []byte) OracleFeed {
	return OracleFeed{
		IsPyth:      oracleType == PYTH,
		IsChainLink: oracleType == CHAINLINK_FEEDS,
		IsMorpho:    oracleType == MORPHO_ORACLE,
		Pyth: lo.Ternary(oracleType == PYTH, func() Pyth {
			var decodedPyth DecodedPyth
			unpacked, err := PythArgument.Unpack(oracleData)
			if err != nil {
				return Pyth{}
			}

			if err := PythArgument.Copy(&decodedPyth, unpacked); err != nil {
				return Pyth{}
			}
			calls := t.ethrpcClient.NewRequest().SetContext(ctx)
			pythStates := make([]DecodedPythStateTuple, len(decodedPyth.FeedIds))
			for j := range decodedPyth.FeedIds {
				calls.AddCall(&ethrpc.Call{
					ABI:    pythABI,
					Target: decodedPyth.Pyth.Hex(),
					Method: "getPriceUnsafe",
					Params: []any{decodedPyth.FeedIds[j]},
				}, []any{&pythStates[j]})
			}
			if _, err := calls.Aggregate(); err != nil {
				return Pyth{}
			}
			return Pyth{
				Pyth: decodedPyth.Pyth,
				FeedIds: lo.Map(decodedPyth.FeedIds, func(item [32]byte, _ int) string {
					return "0x" + hex.EncodeToString(item[:])
				}),
				StalePeriods: decodedPyth.StalePeriods,
				IsMultiplied: decodedPyth.IsMultiplied,
				QuoteType:    decodedPyth.QuoteType,
				PythState: lo.Map(pythStates, func(item DecodedPythStateTuple, _ int) PythState {
					return PythState{
						Price:     uint256.NewInt(uint64(item.Price)),
						Expo:      uint256.NewInt(uint64(item.Expo)),
						Timestamp: uint256.MustFromBig(item.PublishTime),
					}
				}),
				Active: true,
			}
		}, func() Pyth {
			return Pyth{}
		})(),
		Chainlink: lo.Ternary(oracleType == CHAINLINK_FEEDS, func() Chainlink {
			var chainlink Chainlink
			unpacked, err := ChainlinkArgument.Unpack(oracleData)
			if err != nil {
				return Chainlink{}
			}

			if err := ChainlinkArgument.Copy(&chainlink, unpacked); err != nil {
				return Chainlink{}
			}
			calls := t.ethrpcClient.NewRequest().SetContext(ctx)
			chainlinkStates := make([]DecodedChainlink, len(chainlink.CircuitChainlink))
			for j := range chainlink.CircuitChainlink {
				calls.AddCall(&ethrpc.Call{
					ABI:    chainlinkABI,
					Target: chainlink.CircuitChainlink[j].Hex(),
					Method: "latestRoundData",
				}, []any{&chainlinkStates[j]})
			}
			if _, err := calls.Aggregate(); err != nil {
				return Chainlink{}
			}
			chainlink.Answers = lo.Map(chainlinkStates, func(item DecodedChainlink, _ int) *uint256.Int {
				return uint256.MustFromBig(item.Answer)
			})
			chainlink.UpdatedAt = lo.Map(chainlinkStates, func(item DecodedChainlink, _ int) uint64 {
				return item.UpdatedAt.Uint64()
			})
			chainlink.Active = true
			return chainlink
		}, func() Chainlink {
			return Chainlink{}
		})(),
		Max: lo.Ternary(oracleType == MAX, func() *uint256.Int {
			var decodedMax DecodedMax
			unpacked, err := MaxArgument.Unpack(oracleData)
			if err != nil {
				return nil
			}

			if err := MaxArgument.Copy(&decodedMax, unpacked); err != nil {
				return nil
			}
			return uint256.MustFromBig(decodedMax.MaxValue)
		}, func() *uint256.Int {
			return nil
		})(),
		Morpho: lo.Ternary(oracleType == MORPHO_ORACLE, func() Morpho {
			var decodedMorpho DecodedMorpho
			unpacked, err := MorphoArgument.Unpack(oracleData)
			if err != nil {
				return Morpho{}
			}

			if err := MorphoArgument.Copy(&decodedMorpho, unpacked); err != nil {
				return Morpho{}
			}

			// calls := t.ethrpcClient.NewRequest().SetContext(ctx)
			// var baseVault, quoteVault, baseFeed1, baseFeed2, quoteFeed1, quoteFeed2 common.Address
			// var baseVaultConversionSample, quoteVaultConversionSample, scaleFactor *big.Int
			// calls.AddCall(&ethrpc.Call{
			// 	ABI:    morphoABI,
			// 	Target: decodedMorpho.Oracle.Hex(),
			// 	Method: "BASE_VAULT",
			// }, []any{&baseVault}).
			// 	AddCall(&ethrpc.Call{
			// 		ABI:    morphoABI,
			// 		Target: decodedMorpho.Oracle.Hex(),
			// 		Method: "QUOTE_VAULT",
			// 	}, []any{&quoteVault}).
			// 	AddCall(&ethrpc.Call{
			// 		ABI:    morphoABI,
			// 		Target: decodedMorpho.Oracle.Hex(),
			// 		Method: "BASE_FEED_1",
			// 	}, []any{&baseFeed1}).
			// 	AddCall(&ethrpc.Call{
			// 		ABI:    morphoABI,
			// 		Target: decodedMorpho.Oracle.Hex(),
			// 		Method: "BASE_FEED_2",
			// 	}, []any{&baseFeed2}).
			// 	AddCall(&ethrpc.Call{
			// 		ABI:    morphoABI,
			// 		Target: decodedMorpho.Oracle.Hex(),
			// 		Method: "QUOTE_FEED_1",
			// 	}, []any{&quoteFeed1}).
			// 	AddCall(&ethrpc.Call{
			// 		ABI:    morphoABI,
			// 		Target: decodedMorpho.Oracle.Hex(),
			// 		Method: "QUOTE_FEED_2",
			// 	}, []any{&quoteFeed2}).
			// 	AddCall(&ethrpc.Call{
			// 		ABI:    morphoABI,
			// 		Target: decodedMorpho.Oracle.Hex(),
			// 		Method: "BASE_VAULT_CONVERSION_SAMPLE",
			// 	}, []any{&baseVaultConversionSample}).
			// 	AddCall(&ethrpc.Call{
			// 		ABI:    morphoABI,
			// 		Target: decodedMorpho.Oracle.Hex(),
			// 		Method: "QUOTE_VAULT_CONVERSION_SAMPLE",
			// 	}, []any{&quoteVaultConversionSample}).
			// 	AddCall(&ethrpc.Call{
			// 		ABI:    morphoABI,
			// 		Target: decodedMorpho.Oracle.Hex(),
			// 		Method: "SCALE_FACTOR",
			// 	}, []any{&scaleFactor})
			// if _, err := calls.Aggregate(); err != nil {
			// 	return Morpho{}
			// }

			// var baseVaultTotalSupply, baseVaultTotalAssets, quoteVaultTotalSupply, quoteVaultTotalAssets *big.Int
			// calls = t.ethrpcClient.NewRequest().SetContext(ctx)
			// if !account.IsZeroAddress(baseVault) {
			// 	calls.AddCall(&ethrpc.Call{
			// 		ABI:    erc4626ABI,
			// 		Target: baseVault.Hex(),
			// 		Method: "totalSupply",
			// 	}, []any{&baseVaultTotalSupply}).AddCall(&ethrpc.Call{
			// 		ABI:    erc4626ABI,
			// 		Target: baseVault.Hex(),
			// 		Method: "totalAssets",
			// 	}, []any{&baseVaultTotalAssets})
			// }
			// if !account.IsZeroAddress(quoteVault) {
			// 	calls.AddCall(&ethrpc.Call{
			// 		ABI:    erc4626ABI,
			// 		Target: quoteVault.Hex(),
			// 		Method: "totalSupply",
			// 	}, []any{&quoteVaultTotalSupply}).AddCall(&ethrpc.Call{
			// 		ABI:    erc4626ABI,
			// 		Target: quoteVault.Hex(),
			// 		Method: "totalAssets",
			// 	}, []any{&quoteVaultTotalAssets})
			// }
			calls := t.ethrpcClient.NewRequest().SetContext(ctx)
			var price *big.Int
			calls.AddCall(&ethrpc.Call{
				ABI:    morphoABI,
				Target: decodedMorpho.Oracle.Hex(),
				Method: "price",
			}, []any{&price})

			if _, err := calls.Aggregate(); err != nil {
				return Morpho{}
			}
			return Morpho{
				Oracle:              decodedMorpho.Oracle,
				NormalizationFactor: uint256.MustFromBig(decodedMorpho.NormalizationFactor),
				Price:               uint256.MustFromBig(price),
				// BaseVault:                  baseVault,
				// BaseVaultTotalSupply:       uint256.MustFromBig(baseVaultTotalSupply),
				// BaseVaultTotalAssets:       uint256.MustFromBig(baseVaultTotalAssets),
				// QuoteVault:                 quoteVault,
				// QuoteVaultTotalSupply:      uint256.MustFromBig(quoteVaultTotalSupply),
				// QuoteVaultTotalAssets:      uint256.MustFromBig(quoteVaultTotalAssets),
				// BaseFeed1:                  baseFeed1,
				// BaseFeed2:                  baseFeed2,
				// QuoteFeed1:                 quoteFeed1,
				// QuoteFeed2:                 quoteFeed2,
				// BaseVaultConversionSample:  uint256.MustFromBig(baseVaultConversionSample),
				// QuoteVaultConversionSample: uint256.MustFromBig(quoteVaultConversionSample),
				// ScaleFactor:                uint256.MustFromBig(scaleFactor),
				Active: true,
			}
		}, func() Morpho {
			return Morpho{}
		})(),
	}
}
