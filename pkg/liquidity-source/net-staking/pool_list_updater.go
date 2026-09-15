package netstaking

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

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

	netAddr, sNetAddr, blockNumber, err := fetchTokens(ctx, u.ethrpcClient, u.config)
	if err != nil {
		return nil, nil, err
	}

	if u.config.WrapAddress == "" {
		return []entity.Pool{
			{
				Address:   strings.ToLower(u.config.StakingAddress),
				Exchange:  u.config.DexId,
				Type:      DexType,
				Timestamp: time.Now().Unix(),
				Tokens: []*entity.PoolToken{
					{Address: netAddr, Swappable: true},
					{Address: sNetAddr, Swappable: true},
				},
				Reserves:    entity.PoolReserves{"0", "0"},
				BlockNumber: blockNumber,
			},
		}, nil, nil
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

// methodABI builds a synthetic single-method ABI to call a no-arg, address-returning
// view method by name, deriving its 4-byte selector the same way solc does
// (keccak256("name()")[:4]). Lets staking forks that name their base/staked token
// getters differently (net()/sNet() vs nuke()/stakedNuke()) be configured by name
// instead of needing real Go ABI bindings per fork.
func methodABI(methodName string) abi.ABI {
	return abi.ABI{
		Methods: map[string]abi.Method{
			methodName: {
				ID: crypto.Keccak256([]byte(methodName + "()"))[:4],
				Outputs: abi.Arguments{
					{Type: abi.Type{T: abi.AddressTy}},
				},
			},
		},
	}
}

func fetchTokens(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	config *Config,
) (netAddr, sNetAddr string, blockNumber uint64, err error) {
	baseMethod := config.BaseTokenMethod
	if baseMethod == "" {
		baseMethod = defaultBaseTokenMethod
	}
	stakedMethod := config.StakedTokenMethod
	if stakedMethod == "" {
		stakedMethod = defaultStakedTokenMethod
	}

	baseTokenABI := methodABI(baseMethod)
	stakedTokenABI := methodABI(stakedMethod)

	var (
		net  gethcommon.Address
		sNet gethcommon.Address
	)

	req := ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    baseTokenABI,
		Target: config.StakingAddress,
		Method: baseMethod,
	}, []any{&net}).AddCall(&ethrpc.Call{
		ABI:    stakedTokenABI,
		Target: config.StakingAddress,
		Method: stakedMethod,
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
