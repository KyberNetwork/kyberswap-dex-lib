package netstaking

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

	netAddr, sNetAddr, blockNumber, err := fetchTokens(ctx, u.ethrpcClient, u.config.StakingAddress)
	if err != nil {
		return nil, nil, err
	}

	// wsNET is the wrap contract's own ERC20 identity — WrappedStakedNET implements
	// wrap/unwrap and the ERC20 interface on the same contract.
	wsNetAddr := strings.ToLower(u.config.WrapAddress)

	return []entity.Pool{
		{
			Address:   strings.ToLower(u.config.StakingAddress),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: netAddr, Swappable: true},
				{Address: sNetAddr, Swappable: true},
				{Address: wsNetAddr, Swappable: true},
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
) (netAddr, sNetAddr string, blockNumber uint64, err error) {
	var (
		net  gethcommon.Address
		sNet gethcommon.Address
	)

	req := ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    stakingABI,
		Target: stakingAddress,
		Method: "net",
	}, []any{&net}).AddCall(&ethrpc.Call{
		ABI:    stakingABI,
		Target: stakingAddress,
		Method: "sNet",
	}, []any{&sNet})

	resp, err := req.Aggregate()
	if err != nil {
		return "", "", 0, err
	}
	if resp.BlockNumber == nil {
		blockNumber = 0
	} else {
		blockNumber = resp.BlockNumber.Uint64()
	}

	return hexutil.Encode(net[:]), hexutil.Encode(sNet[:]), blockNumber, nil
}
