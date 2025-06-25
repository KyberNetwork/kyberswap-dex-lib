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
	redemptionFees := make([]DecodedRedemptionFees, len(collateralList))
	collateralWhitelistData := make([][]byte, len(collateralList))
	var totalStablecoinIssued *big.Int
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
			Method: "getRedemptionFees",
			Params: nil,
		}, []any{&redemptionFees[i]})
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
		Method: "getTotalIssued",
		Params: nil,
	}, []any{&totalStablecoinIssued})

	if _, err := calls.Aggregate(); err != nil {
		return p, err
	}

	transmuterState := TransmuterState{}
	for i := range collateralList {
		transmuterState.XRedemptionCurve = redemptionFees[i].XRedemptionCurve
		transmuterState.YRedemptionCurve = redemptionFees[i].YRedemptionCurve
		transmuterState.TotalStablecoinIssued = uint256.MustFromBig(totalStablecoinIssued)
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
				// ExternalOracle: collateralConfigs[i].ExternalOracle, // TODO:
				OracleFeed: OracleFeed{
					IsPyth:      OracleReadType(collateralConfigs[i].OracleType) == PYTH,
					IsChainLink: OracleReadType(collateralConfigs[i].OracleType) == CHAINLINK_FEEDS,
					IsMorpho:    OracleReadType(collateralConfigs[i].OracleType) == MORPHO_ORACLE,
					Pyth: lo.Ternary(OracleReadType(collateralConfigs[i].OracleType) == PYTH, func() Pyth {
						var decodedPyth DecodedPyth
						unpacked, err := PythArgument.Unpack(collateralConfigs[i].OracleData)
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
					Chainlink: lo.Ternary(OracleReadType(collateralConfigs[i].OracleType) == CHAINLINK_FEEDS, func() Chainlink {
						var chainlink Chainlink
						unpacked, err := ChainlinkArgument.Unpack(collateralConfigs[i].OracleData)
						if err != nil {
							return Chainlink{}
						}

						if err := ChainlinkArgument.Copy(&chainlink, unpacked); err != nil {
							return Chainlink{}
						}
						return chainlink
					}, func() Chainlink {
						return Chainlink{}
					})(),
				},
			},
		}
	}

	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
