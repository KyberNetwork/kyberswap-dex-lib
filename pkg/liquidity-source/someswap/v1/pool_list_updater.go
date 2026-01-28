package someswapv1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}
)

type feeBase struct {
	BaseFeeBps *big.Int `abi:"baseFeeBps"`
	WToken0In  *big.Int `abi:"wToken0In"`
	WToken1In  *big.Int `abi:"wToken1In"`
}

type dynamicFee struct {
	CurrentBps  *big.Int `abi:"currentBps"`
	Initialized uint8    `abi:"initialized"`
}

type feeParams struct {
	BaseFee          feeBase    `abi:"baseFee"`
	DynamicFee       dynamicFee `abi:"dynamicFee"`
	ProtocolShareBps *big.Int   `abi:"protocolShareBps"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = DexType
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"exchange": dexID}).Info("Started getting new pools")

	allPairsLength, err := u.getAllPairsLength(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID}).Error("getAllPairsLength failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Warn("getOffset failed")
	}

	batchSize := u.getBatchSize(allPairsLength, u.config.NewPoolLimit, offset)
	if batchSize == 0 {
		return nil, metadataBytes, nil
	}

	pairAddresses, err := u.listPairAddresses(ctx, offset, batchSize)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("listPairAddresses failed")
		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, pairAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("newMetadata failed")
		return nil, metadataBytes, err
	}

	logger.WithFields(
		logger.Fields{
			"dex_id":      dexID,
			"valid_pools": len(pools),
			"offset":      offset,
			"duration_ms": time.Since(startTime).Milliseconds(),
		},
	).Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	var allPairsLength *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodAllPairsLength,
	}, []any{&allPairsLength})

	if _, err := req.Call(); err != nil {
		return 0, err
	}
	return int(allPairsLength.Int64()), nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}
	var metadata Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}
	return metadata.Offset, nil
}

func (u *PoolsListUpdater) getBatchSize(totalPools, limit, offset int) int {
	if offset >= totalPools || limit == 0 {
		return 0
	}
	batchSize := limit
	if offset+batchSize > totalPools {
		batchSize = totalPools - offset
	}
	return batchSize
}

func (u *PoolsListUpdater) listPairAddresses(ctx context.Context, offset int, batchSize int) ([]common.Address, error) {
	listPairAddressesResult := make([]common.Address, batchSize)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i := 0; i < batchSize; i++ {
		index := big.NewInt(int64(offset + i))
		req.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodGetPair,
			Params: []any{index},
		}, []any{&listPairAddressesResult[i]})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}

	var pairAddresses []common.Address
	for i, isSuccess := range resp.Result {
		if !isSuccess {
			continue
		}
		pairAddresses = append(pairAddresses, listPairAddressesResult[i])
	}
	return pairAddresses, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	token0List, token1List, fees, err := u.listPairData(ctx, pairAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, len(pairAddresses))
	for i, pairAddress := range pairAddresses {
		baseFeeBps, _ := fees[i].BaseFee.BaseFeeBps.Float64()
		dynamicFeeBps, _ := fees[i].DynamicFee.CurrentBps.Float64()
		swapFee := min(maxFeeBps, baseFeeBps+dynamicFeeBps) / bps

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			WTokens: [2]*uint256.Int{toU256(fees[i].BaseFee.WToken0In), toU256(fees[i].BaseFee.WToken1In)},
		})

		pools[i] = entity.Pool{
			Address:   hexutil.Encode(pairAddress[:]),
			SwapFee:   swapFee,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(token0List[i][:]), Swappable: true},
				{Address: hexutil.Encode(token1List[i][:]), Swappable: true},
			},
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}

func (u *PoolsListUpdater) listPairData(ctx context.Context, pairAddresses []common.Address) ([]common.Address,
	[]common.Address, []feeParams, error) {
	token0List := make([]common.Address, len(pairAddresses))
	token1List := make([]common.Address, len(pairAddresses))
	feeParamsArr := make([]feeParams, len(pairAddresses))

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, pairAddress := range pairAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddress.Hex(),
			Method: pairMethodToken0,
		}, []any{&token0List[i]}).AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddress.Hex(),
			Method: pairMethodToken1,
		}, []any{&token1List[i]}).AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddress.Hex(),
			Method: pairMethodFee,
		}, []any{&feeParamsArr[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, nil, nil, err
	}

	return token0List, token1List, feeParamsArr, nil
}

func (u *PoolsListUpdater) newMetadata(offset int) ([]byte, error) {
	return json.Marshal(Metadata{Offset: offset})
}

func toU256(b *big.Int) *uint256.Int {
	u, overflow := uint256.FromBig(b)
	if u == nil || overflow {
		return new(uint256.Int)
	}
	return u
}
