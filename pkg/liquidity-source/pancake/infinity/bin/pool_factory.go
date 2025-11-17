package bin

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/bin/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type PoolFactory struct {
	config              *Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			shared.BinPoolManagerABI.Events["Initialize"].ID: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	p, err := shared.BinPoolManagerFilterer.ParseInitialize(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abi.PancakeInfinityPoolManagerInitialize, blockNbr uint64) (*entity.Pool, error) {
	chainId := valueobject.ChainID(f.config.ChainID)

	swapFee, _ := p.Fee.Float64()

	params := p.Parameters[:]
	binStep := GetBinStep(params)
	hasSwapPermissions := shared.HasSwapPermissions(params)

	extraBytes, _ := json.Marshal(Extra{
		ActiveBinID: uint32(p.ActiveId.Uint64()),
	})
	staticExtra := StaticExtra{
		HasSwapPermissions: hasSwapPermissions,
		Fee:                uint32(p.Fee.Uint64()),
		IsNative:           [2]bool{eth.IsZeroAddress(p.Currency0), eth.IsZeroAddress(p.Currency1)},
		Parameters:         hexutil.Encode(params),
		BinStep:            binStep,
		PoolManagerAddress: common.HexToAddress(f.config.BinPoolManagerAddress),
		HooksAddress:       p.Hooks,
		Permit2Address:     common.HexToAddress(f.config.Permit2Address),
		VaultAddress:       common.HexToAddress(f.config.VaultAddress),
		Multicall3Address:  common.HexToAddress(f.config.Multicall3Address),
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	hook, _ := GetHook(staticExtra.HooksAddress)
	return &entity.Pool{
		Address:   hexutil.Encode(p.Id[:]),
		SwapFee:   swapFee,
		Exchange:  hook.GetExchange(),
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: currencyToToken(p.Currency0, chainId), Swappable: true},
			{Address: currencyToToken(p.Currency1, chainId), Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNbr,
	}, nil
}

func currencyToToken(currency common.Address, chainId valueobject.ChainID) string {
	if eth.IsZeroAddress(currency) {
		return strings.ToLower(valueobject.WrappedNativeMap[chainId])
	}
	return hexutil.Encode(currency[:])
}

func DecodePoolAddress(log ethtypes.Log) (string, error) {
	if len(log.Topics) == 0 || eth.IsZeroAddress(log.Address) {
		return "", nil
	}

	switch log.Topics[0] {
	case shared.BinPoolManagerABI.Events["Initialize"].ID,
		shared.BinPoolManagerABI.Events["Donate"].ID,
		shared.BinPoolManagerABI.Events["Mint"].ID,
		shared.BinPoolManagerABI.Events["Burn"].ID,
		shared.BinPoolManagerABI.Events["ProtocolFeeUpdated"].ID,
		shared.BinPoolManagerABI.Events["DynamicLPFeeUpdated"].ID,
		shared.BinPoolManagerABI.Events["Swap"].ID: // these events have the pool address in topic1
		if len(log.Topics) < 2 {
			break
		}
		return hexutil.Encode(log.Topics[1][:]), nil
	}

	return "", nil
}
