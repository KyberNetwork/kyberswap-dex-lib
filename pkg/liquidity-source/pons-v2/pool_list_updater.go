package ponsv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolsListUpdaterMetadata tracks the last fully-scanned block. Factory.
// TokenLaunched is the only discovery surface (no allLaunchedTokens-style
// enumerator exists on PonsV2LaunchFactory), so discovery is a from-block
// eth_getLogs scan rather than an index/view-based walk.
type PoolsListUpdaterMetadata struct {
	LastScannedBlock uint64 `json:"lastScannedBlock"`
}

// tokenLaunchedLog mirrors Factory.TokenLaunched's decoded fields (only the
// ones this integration needs; deployer and launchConfigId are dropped).
type tokenLaunchedLog struct {
	Token     common.Address
	Curve     common.Address
	PairToken common.Address
}

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

// maxBlockRangePerScan caps a single eth_getLogs call's block span. Public
// RPC providers commonly cap getLogs windows (often 1-10k blocks); this
// value should be tuned against the Robinhood-chain RPC's actual limit
// during dex-verify.
const maxBlockRangePerScan = 5_000

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
		logger:       logger.WithFields(logger.Fields{"dex_id": cfg.DexID}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()

	fromBlock, err := u.getFromBlock(metadataBytes)
	if err != nil {
		return nil, metadataBytes, err
	}

	head, err := u.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}
	if fromBlock > head {
		return nil, metadataBytes, nil
	}

	toBlock := min(fromBlock+maxBlockRangePerScan-1, head)

	logs, err := u.ethrpcClient.GetETHClient().FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{common.HexToAddress(u.config.Factory)},
		Topics:    [][]common.Hash{{tokenLaunchedEventHash}},
	})
	if err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Error("failed to filter TokenLaunched logs")
		return nil, metadataBytes, err
	}

	launches := make([]tokenLaunchedLog, 0, len(logs))
	for _, l := range logs {
		decoded, err := decodeTokenLaunched(l)
		if err != nil {
			u.logger.WithFields(logger.Fields{"error": err, "txHash": l.TxHash.Hex()}).
				Error("failed to decode TokenLaunched log")
			continue
		}
		launches = append(launches, decoded)
	}

	pools, err := u.buildPools(ctx, launches)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{LastScannedBlock: toBlock + 1})
	if err != nil {
		return nil, metadataBytes, err
	}

	u.logger.WithFields(logger.Fields{
		"newPools":   len(pools),
		"fromBlock":  fromBlock,
		"toBlock":    toBlock,
		"durationMs": time.Since(startTime).Milliseconds(),
	}).Info("finished scanning for new pons-v2 curves")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getFromBlock(metadataBytes []byte) (uint64, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}
	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}
	return metadata.LastScannedBlock, nil
}

// decodeTokenLaunched decodes topics positionally (token, curve, deployer
// are indexed; pairToken/launchConfigId/graduationThreshold are the
// non-indexed data payload, in that declaration order) rather than by
// reflect-based field-name matching, to avoid ABI-unpack naming pitfalls.
func decodeTokenLaunched(l types.Log) (tokenLaunchedLog, error) {
	if len(l.Topics) != 4 {
		return tokenLaunchedLog{}, ErrInvalidToken
	}

	values, err := factoryABI.Events["TokenLaunched"].Inputs.NonIndexed().Unpack(l.Data)
	if err != nil {
		return tokenLaunchedLog{}, err
	}
	if len(values) == 0 {
		return tokenLaunchedLog{}, ErrInvalidToken
	}
	pairToken, ok := values[0].(common.Address)
	if !ok {
		return tokenLaunchedLog{}, ErrInvalidToken
	}

	return tokenLaunchedLog{
		Token:     common.BytesToAddress(l.Topics[1].Bytes()),
		Curve:     common.BytesToAddress(l.Topics[2].Bytes()),
		PairToken: pairToken,
	}, nil
}

// buildPools fetches each newly-launched curve's immutable config
// (feeBps/creatorTaxBps/reservedTokens/isNativeQuote) in one multicall batch
// and skips curves that have already graduated by the time discovery runs.
func (u *PoolsListUpdater) buildPools(ctx context.Context, launches []tokenLaunchedLog) ([]entity.Pool, error) {
	if len(launches) == 0 {
		return nil, nil
	}

	n := len(launches)
	feeBpsList := make([]*big.Int, n)
	creatorTaxBpsList := make([]*big.Int, n)
	reservedTokensList := make([]*big.Int, n)
	isNativeQuoteList := make([]bool, n)
	graduatedList := make([]bool, n)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, launch := range launches {
		curve := launch.Curve.Hex()
		req.AddCall(&ethrpc.Call{ABI: curveABI, Target: curve, Method: curveMethodFeeBps}, []any{&feeBpsList[i]})
		req.AddCall(&ethrpc.Call{ABI: curveABI, Target: curve, Method: curveMethodCreatorTax}, []any{&creatorTaxBpsList[i]})
		req.AddCall(&ethrpc.Call{ABI: curveABI, Target: curve, Method: curveMethodReserved}, []any{&reservedTokensList[i]})
		req.AddCall(&ethrpc.Call{ABI: curveABI, Target: curve, Method: curveMethodIsNative}, []any{&isNativeQuoteList[i]})
		req.AddCall(&ethrpc.Call{ABI: curveABI, Target: curve, Method: curveMethodGraduated}, []any{&graduatedList[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	wrappedNative := common.HexToAddress(valueobject.WrappedNativeMap[u.config.ChainID])

	pools := make([]entity.Pool, 0, n)
	for i, launch := range launches {
		if graduatedList[i] {
			continue
		}

		isNativeQuote := isNativeQuoteList[i]
		quoteAddress := launch.PairToken
		if isNativeQuote {
			quoteAddress = wrappedNative
		}

		staticExtra := StaticExtra{
			FeeBps:         uint16(feeBpsList[i].Uint64()),
			CreatorTaxBps:  uint16(creatorTaxBpsList[i].Uint64()),
			ReservedTokens: uint256.MustFromBig(reservedTokensList[i]),
			IsNativeQuote:  isNativeQuote,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(launch.Curve[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(quoteAddress[:]), Swappable: true},
				{Address: hexutil.Encode(launch.Token[:]), Swappable: true},
			},
			StaticExtra: string(staticExtraBytes),
		})
	}

	return pools, nil
}
