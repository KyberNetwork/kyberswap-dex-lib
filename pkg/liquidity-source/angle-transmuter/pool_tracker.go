package angletransmuter

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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

type DecodedMax struct {
	MaxValue *big.Int
}

type DecodedMorpho struct {
	Oracle              common.Address
	NormalizationFactor *big.Int
}

type ManagerData struct {
	SubCollaterals []common.Address
	Config         []byte
}

type CollateralInfo struct {
	IsManaged         uint8
	IsMintLive        uint8
	IsBurnLive        uint8
	Decimals          uint8
	OnlyWhitelisted   uint8
	NormalizedStables *big.Int
	XFeeMint          []uint64
	YFeeMint          []int64
	XFeeBurn          []uint64
	YFeeBurn          []int64
	OracleConfig      []byte
	WhitelistData     []byte
	ManagerData       ManagerData
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
	var collateralList []common.Address
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getCollateralList",
		}, []any{&collateralList}).Aggregate(); err != nil {
		return p, err
	}

	collateralInfo := make([]*CollateralInfo, len(collateralList))
	oracleConfigs := make([]DecodedOracleConfig, len(collateralList))
	issuedByCollateral := make([]DecodedIssuedByCollateral, len(collateralList))
	stablecoinCap := make([]*big.Int, len(collateralList))
	collateralBalances := make([]*big.Int, len(collateralList))
	var totalStablecoinIssued *big.Int

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	for i, collateral := range collateralList {
		collateralInfo[i] = &CollateralInfo{}
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getCollateralInfo",
			Params: []any{collateral},
		}, []any{&collateralInfo[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getOracle",
			Params: []any{collateral},
		}, []any{&oracleConfigs[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getIssuedByCollateral",
			Params: []any{collateral},
		}, []any{&issuedByCollateral[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    transmuterABI,
			Target: t.config.Transmuter,
			Method: "getStablecoinCap",
			Params: []any{collateral},
		}, []any{&stablecoinCap[i]})

		// For unmanaged collateral tokens only
		calls.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: collateral.String(),
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(t.config.Transmuter)},
		}, []any{&collateralBalances[i]})
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    transmuterABI,
		Target: t.config.Transmuter,
		Method: "getTotalIssued",
	}, []any{&totalStablecoinIssued})

	if _, err := calls.TryAggregate(); err != nil {
		return p, err
	}

	exchange := valueobject.Exchange(p.Exchange)
	// Convert on-chain oracle types to our enum
	oracleConfigs = lo.Map(oracleConfigs, func(cfg DecodedOracleConfig, _ int) DecodedOracleConfig {
		cfg.OracleType = uint8(convertOracleType(exchange, cfg.OracleType))
		cfg.TargetType = uint8(convertOracleType(exchange, cfg.TargetType))
		return cfg
	})

	transmuterState := TransmuterState{
		Collaterals:           make(map[string]CollateralState),
		TotalStablecoinIssued: uint256.MustFromBig(totalStablecoinIssued),
	}

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

	calls = t.ethrpcClient.NewRequest().SetContext(ctx)
	for i, collat := range oracleConfigs {
		configs := []struct {
			typ  OracleReadType
			data []byte
		}{
			{OracleReadType(collat.OracleType), collat.OracleData},
			{OracleReadType(collat.TargetType), collat.TargetData},
		}

		for j, cfg := range configs {
			switch cfg.typ {
			case PYTH:
				var decodedPyth DecodedPyth
				unpacked, err := PythArgument.Unpack(cfg.data)
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
				unpacked, err := ChainlinkArgument.Unpack(cfg.data)
				if err != nil {
					return p, err
				}

				if err := ChainlinkArgument.Copy(&chainlink, unpacked); err != nil {
					return p, err
				}

				chainlinks[j][i] = chainlink
				chainlinks[j][i].RawStates = make([]DecodedChainlink, len(chainlink.CircuitChainlink))
				for k := range chainlink.CircuitChainlink {
					calls.AddCall(&ethrpc.Call{
						ABI:    chainlinkABI,
						Target: chainlink.CircuitChainlink[k].Hex(),
						Method: "latestRoundData",
					}, []any{&chainlinks[j][i].RawStates[k]})
				}
			case MORPHO_ORACLE:
				var decodedMorpho DecodedMorpho
				unpacked, err := MorphoArgument.Unpack(cfg.data)
				if err != nil {
					return p, err
				}

				if err := MorphoArgument.Copy(&decodedMorpho, unpacked); err != nil {
					return p, err
				}

				morphos[j][i] = Morpho{
					Oracle:              decodedMorpho.Oracle,
					NormalizationFactor: uint256.MustFromBig(decodedMorpho.NormalizationFactor),
				}

				calls.AddCall(&ethrpc.Call{
					ABI:    morphoABI,
					Target: decodedMorpho.Oracle.Hex(),
					Method: "price",
				}, []any{&morphos[j][i].RawState})
			case MAX:
				var decodedMax DecodedMax
				unpacked, err := MaxArgument.Unpack(cfg.data)
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
		collatInfo := collateralInfo[i]
		oracleCfg := oracleConfigs[i]
		transmuterState.Collaterals[hexutil.Encode(collateralList[i][:])] = CollateralState{
			IsManaged:         collatInfo.IsManaged != 0,
			IsBurnLive:        collatInfo.IsBurnLive != 0,
			IsMintLive:        collatInfo.IsMintLive != 0,
			Balance:           uint256.MustFromBig(collateralBalances[i]),
			NormalizedStables: uint256.MustFromBig(collatInfo.NormalizedStables),
			Fees: Fees{
				XFeeMint: lo.Map(collatInfo.XFeeMint, func(item uint64, _ int) *uint256.Int {
					return uint256.NewInt(item)
				}),
				YFeeMint: lo.Map(collatInfo.YFeeMint, func(item int64, _ int) *uint256.Int {
					return u256.MustFromInt64(item)
				}),
				XFeeBurn: lo.Map(collatInfo.XFeeBurn, func(item uint64, _ int) *uint256.Int {
					return uint256.NewInt(item)
				}),
				YFeeBurn: lo.Map(collatInfo.YFeeBurn, func(item int64, _ int) *uint256.Int {
					return u256.MustFromInt64(item)
				}),
			},
			StablecoinsFromCollateral: uint256.MustFromBig(issuedByCollateral[i].StablecoinsFromCollateral),
			StablecoinsIssued:         uint256.MustFromBig(issuedByCollateral[i].StablecoinsIssued),
			StablecoinCap:             uint256.MustFromBig(stablecoinCap[i]),
			Config: Oracle{
				OracleType: OracleReadType(oracleConfigs[i].OracleType),
				TargetType: OracleReadType(oracleConfigs[i].TargetType),
				OracleFeed: t.getOracleFeed(0, i, oracleCfg, pyths, chainlinks, morphos, maxes),
				TargetFeed: t.getOracleFeed(1, i, oracleCfg, pyths, chainlinks, morphos, maxes),
				Hyperparameters: func() Hyperparameters {
					unpacked, err := HyperparametersArgument.Unpack(oracleCfg.Hyperparameters)
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

	extraBytes, err := json.Marshal(Extra{Transmuter: transmuterState})
	if err != nil {
		logger.WithFields(klog.Fields{"error": err}).Error("failed to marshal extra data")
		return p, err
	}
	p.BlockNumber = res.BlockNumber.Uint64()
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Tokens = append(lo.Map(collateralList, func(token common.Address, _ int) *entity.PoolToken {
		return &entity.PoolToken{
			Address:   hexutil.Encode(token[:]),
			Swappable: true,
		}
	}), p.Tokens[len(p.Tokens)-1]) // last one is stable token
	p.Reserves = append(lo.Map(collateralBalances, func(b *big.Int, _ int) string {
		return b.String()
	}), defaultReserve)

	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

func (t *PoolTracker) getOracleFeed(oracleOrTarget int, index int, oracleCfg DecodedOracleConfig,
	pyths [2][]Pyth, chainlinks [2][]Chainlink, morphos [2][]Morpho, maxes [2][]*uint256.Int) OracleFeed {
	oracleType := OracleReadType(lo.Ternary(oracleOrTarget == 0, oracleCfg.OracleType, oracleCfg.TargetType))
	return OracleFeed{
		IsPyth:      oracleType == PYTH,
		IsChainLink: oracleType == CHAINLINK_FEEDS,
		IsMorpho:    oracleType == MORPHO_ORACLE,
		Pyth: lo.Ternary(oracleType == PYTH, func() *Pyth {
			pyths[oracleOrTarget][index].PythState = lo.Map(pyths[oracleOrTarget][index].RawStates,
				func(item DecodedPythStateTuple, _ int) PythState {
					return PythState{
						Price:     uint256.NewInt(uint64(item.Price)),
						Expo:      uint256.MustFromBig(big.NewInt(int64(item.Expo))),
						Timestamp: uint256.MustFromBig(item.PublishTime),
					}
				})
			return &pyths[oracleOrTarget][index]
		}, func() *Pyth {
			return nil
		})(),
		Chainlink: lo.Ternary(oracleType == CHAINLINK_FEEDS, func() *Chainlink {
			chainlinks[oracleOrTarget][index].Answers = lo.Map(chainlinks[oracleOrTarget][index].RawStates,
				func(item DecodedChainlink, _ int) *uint256.Int {
					return uint256.MustFromBig(item.Answer)
				})
			chainlinks[oracleOrTarget][index].UpdatedAt = lo.Map(chainlinks[oracleOrTarget][index].RawStates,
				func(item DecodedChainlink, _ int) uint64 {
					return item.UpdatedAt.Uint64()
				})
			return &chainlinks[oracleOrTarget][index]
		}, func() *Chainlink {
			return nil
		})(),
		Morpho: lo.Ternary(oracleType == MORPHO_ORACLE, func() *Morpho {
			morphos[oracleOrTarget][index].Price = uint256.MustFromBig(morphos[oracleOrTarget][index].RawState)
			return &morphos[oracleOrTarget][index]
		}, func() *Morpho {
			return nil
		})(),
		Max: lo.Ternary(oracleType == MAX, func() *uint256.Int {
			return maxes[oracleOrTarget][index]
		}, func() *uint256.Int {
			return nil
		})(),
	}
}
