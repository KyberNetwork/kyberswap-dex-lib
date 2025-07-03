package angletransmuter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
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

	calls = t.ethrpcClient.NewRequest().SetContext(ctx)
	pyths := [2][]Pyth{
		make([]Pyth, len(collateralList)), // oracle
		make([]Pyth, len(collateralList)), // target
	}
	chainlinks := [2][]Chainlink{
		make([]Chainlink, len(collateralList)), // oracle
		make([]Chainlink, len(collateralList)), // target
	}
	morphos := [2][]Morpho{
		make([]Morpho, len(collateralList)), // oracle
		make([]Morpho, len(collateralList)), // target
	}
	maxes := [2][]*uint256.Int{
		make([]*uint256.Int, len(collateralList)), // oracle
		make([]*uint256.Int, len(collateralList)), // target
	}

	for i, collat := range collateralConfigs {
		for j, typee := range []OracleReadType{OracleReadType(collat.OracleType), OracleReadType(collat.TargetType)} {
			switch typee {
			case PYTH:
				var decodedPyth DecodedPyth
				unpacked, err := PythArgument.Unpack(lo.Ternary(j == 0, collat.OracleData, collat.TargetData))
				if err != nil {
					return p, err
				}
				if err := PythArgument.Copy(&decodedPyth, unpacked); err != nil {
					return p, err
				}
				pyths[j][i] = Pyth{
					Pyth: decodedPyth.Pyth,
					FeedIds: lo.Map(decodedPyth.FeedIds, func(item [32]byte, _ int) string {
						return "0x" + hex.EncodeToString(item[:])
					}),
					StalePeriods: decodedPyth.StalePeriods,
					IsMultiplied: decodedPyth.IsMultiplied,
					QuoteType:    decodedPyth.QuoteType,
					RawStates:    make([]DecodedPythStateTuple, len(decodedPyth.FeedIds)),
					Active:       true,
				}
				for k := range decodedPyth.FeedIds {
					calls.AddCall(&ethrpc.Call{
						ABI:    pythABI,
						Target: decodedPyth.Pyth.Hex(),
						Method: "getPriceUnsafe",
						Params: []any{decodedPyth.FeedIds[k]},
					}, []any{&pyths[j][i].RawStates[k]})
				}
			case CHAINLINK_FEEDS:
				var chainlink Chainlink
				unpacked, err := ChainlinkArgument.Unpack(lo.Ternary(j == 0, collat.OracleData, collat.TargetData))
				if err != nil {
					return p, err
				}

				if err := ChainlinkArgument.Copy(&chainlink, unpacked); err != nil {
					return p, err
				}

				chainlinks[j][i] = chainlink
				chainlinks[j][i].RawStates = make([]DecodedChainlink, len(chainlink.CircuitChainlink))
				chainlinks[j][i].Active = true
				for k := range chainlink.CircuitChainlink {
					calls.AddCall(&ethrpc.Call{
						ABI:    chainlinkABI,
						Target: chainlink.CircuitChainlink[k].Hex(),
						Method: "latestRoundData",
					}, []any{&chainlinks[j][i].RawStates[k]})
				}
			case MORPHO_ORACLE:
				var decodedMorpho DecodedMorpho
				unpacked, err := MorphoArgument.Unpack(lo.Ternary(j == 0, collat.OracleData, collat.TargetData))
				if err != nil {
					return p, err
				}

				if err := MorphoArgument.Copy(&decodedMorpho, unpacked); err != nil {
					return p, err
				}

				morphos[j][i] = Morpho{
					Oracle:              decodedMorpho.Oracle,
					NormalizationFactor: uint256.MustFromBig(decodedMorpho.NormalizationFactor),
					Active:              true,
				}

				calls.AddCall(&ethrpc.Call{
					ABI:    morphoABI,
					Target: decodedMorpho.Oracle.Hex(),
					Method: "price",
				}, []any{&morphos[j][i].RawState})
			case MAX:
				var decodedMax DecodedMax
				unpacked, err := MaxArgument.Unpack(lo.Ternary(j == 0, collat.OracleData, collat.TargetData))
				if err != nil {
					return p, err
				}

				if err := MaxArgument.Copy(&decodedMax, unpacked); err != nil {
					return p, err
				}
				maxes[j][i] = uint256.MustFromBig(decodedMax.MaxValue)
			}
		}
	}

	res, err := calls.Aggregate()
	if err != nil {
		return p, err
	}

	for i := range collateralList {
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
				OracleFeed: t.getOracleFeed(0, i, collateralConfigs[i], pyths, chainlinks, morphos, maxes),
				TargetFeed: t.getOracleFeed(1, i, collateralConfigs[i], pyths, chainlinks, morphos, maxes),
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

	extra := Extra{
		Transmuter: transmuterState,
		Gas: Gas{
			Mint: 400000,
			Burn: 450000,
		},
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(klog.Fields{"error": err}).Error("failed to marshal extra data")
		return p, err
	}
	p.BlockNumber = res.BlockNumber.Uint64()
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	tokens := lo.Map(collateralList, func(token common.Address, _ int) *entity.PoolToken {
		return &entity.PoolToken{
			Address:   strings.ToLower(token.Hex()),
			Swappable: true,
		}
	})
	p.Tokens = append(tokens, p.Tokens[len(p.Tokens)-1]) // last one is agToken
	p.Reserves = lo.Map(p.Tokens, func(token *entity.PoolToken, _ int) string {
		return abi.MaxUint256.String()
	})
	return p, nil
}

func (t *PoolTracker) getOracleFeed(oracleOrTarget int, index int, decodedOracleConfig DecodedOracleConfig, pyths [2][]Pyth, chainlinks [2][]Chainlink, morphos [2][]Morpho, maxes [2][]*uint256.Int) OracleFeed {
	oracleType := OracleReadType(lo.Ternary(oracleOrTarget == 0, decodedOracleConfig.OracleType, decodedOracleConfig.TargetType))
	return OracleFeed{
		IsPyth:      oracleType == PYTH,
		IsChainLink: oracleType == CHAINLINK_FEEDS,
		IsMorpho:    oracleType == MORPHO_ORACLE,
		Pyth: lo.Ternary(oracleType == PYTH, func() Pyth {
			pyths[oracleOrTarget][index].PythState = lo.Map(pyths[oracleOrTarget][index].RawStates, func(item DecodedPythStateTuple, _ int) PythState {
				return PythState{
					Price:     uint256.NewInt(uint64(item.Price)),
					Expo:      uint256.MustFromBig(big.NewInt(int64(item.Expo))),
					Timestamp: uint256.MustFromBig(item.PublishTime),
				}
			})
			return pyths[oracleOrTarget][index]
		}, func() Pyth {
			return Pyth{}
		})(),
		Chainlink: lo.Ternary(oracleType == CHAINLINK_FEEDS, func() Chainlink {
			chainlinks[oracleOrTarget][index].Answers = lo.Map(chainlinks[oracleOrTarget][index].RawStates, func(item DecodedChainlink, _ int) *uint256.Int {
				return uint256.MustFromBig(item.Answer)
			})
			chainlinks[oracleOrTarget][index].UpdatedAt = lo.Map(chainlinks[oracleOrTarget][index].RawStates, func(item DecodedChainlink, _ int) uint64 {
				return item.UpdatedAt.Uint64()
			})
			return chainlinks[oracleOrTarget][index]
		}, func() Chainlink {
			return Chainlink{}
		})(),
		Morpho: lo.Ternary(oracleType == MORPHO_ORACLE, func() Morpho {
			morphos[oracleOrTarget][index].Price = uint256.MustFromBig(morphos[oracleOrTarget][index].RawState)
			return morphos[oracleOrTarget][index]
		}, func() Morpho {
			return Morpho{}
		})(),
		Max: lo.Ternary(oracleType == MAX, func() *uint256.Int {
			return maxes[oracleOrTarget][index]
		}, func() *uint256.Int {
			return nil
		})(),
	}
}
