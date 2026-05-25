package gohm

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	u.hasInitialized = true

	ohmAddr, sohmAddr, gohmAddr, blockNumber, err := fetchTokens(ctx, u.ethrpcClient, u.config.StakingAddress)
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{
		{
			Address:   strings.ToLower(u.config.StakingAddress),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: ohmAddr, Swappable: true},
				{Address: sohmAddr, Swappable: true},
				{Address: gohmAddr, Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0", "0"},
			BlockNumber: blockNumber,
		},
	}, nil, nil
}

func fetchTokens(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	stakingAddress string,
) (ohmAddr, sohmAddr, gohmAddr string, blockNumber uint64, err error) {
	var (
		ohm  gethcommon.Address
		sohm gethcommon.Address
		gohm gethcommon.Address
	)

	req := ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    olympusStakingABI,
		Target: stakingAddress,
		Method: "OHM",
	}, []any{&ohm}).AddCall(&ethrpc.Call{
		ABI:    olympusStakingABI,
		Target: stakingAddress,
		Method: "sOHM",
	}, []any{&sohm}).AddCall(&ethrpc.Call{
		ABI:    olympusStakingABI,
		Target: stakingAddress,
		Method: "gOHM",
	}, []any{&gohm})

	resp, err := req.Aggregate()
	if err != nil {
		return "", "", "", 0, err
	}
	if resp.BlockNumber == nil {
		blockNumber = 0
	} else {
		blockNumber = resp.BlockNumber.Uint64()
	}

	return hexutil.Encode(ohm[:]), hexutil.Encode(sohm[:]), hexutil.Encode(gohm[:]), blockNumber, nil
}
