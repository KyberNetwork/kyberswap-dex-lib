package liquidityparty

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestPoolFactory_DecodePoolCreated(t *testing.T) {
	f := NewPoolFactory(&Config{DexID: DexType})

	poolAddr := common.HexToAddress("0x1270Da05Cf1d047763CEEfDe25a4a5438b26fdA6")
	tokens := []common.Address{
		common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), // USDC
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), // WETH
		common.HexToAddress("0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9"), // AAVE
	}

	// Pack the non-indexed args (name, symbol, tokens[]) exactly as the event emits them.
	data, err := partyPlannerABI.Events[plannerEventPartyStarted].Inputs.NonIndexed().
		Pack("Party", "PARTY", tokens)
	require.NoError(t, err)

	log := ethtypes.Log{
		Address: common.HexToAddress(mainnetConfig.PartyPlannerAddress),
		Topics: []common.Hash{
			partyStartedEventTopic,
			common.BytesToHash(poolAddr.Bytes()), // indexed pool
		},
		Data:        data,
		BlockNumber: 25301966,
	}

	require.True(t, f.IsEventSupported(partyStartedEventTopic))
	require.False(t, f.IsEventSupported(killedEventTopic))

	p, err := f.DecodePoolCreated(log)
	require.NoError(t, err)
	require.Equal(t, strings.ToLower(poolAddr.Hex()), p.Address, "pool address lowercased")
	require.Equal(t, DexType, p.Type)
	require.Equal(t, DexType, p.Exchange)
	require.EqualValues(t, 25301966, p.BlockNumber)
	require.Len(t, p.Tokens, len(tokens))
	require.Len(t, p.Reserves, len(tokens))
	for i, tok := range p.Tokens {
		require.Equal(t, strings.ToLower(tokens[i].Hex()), tok.Address, "token lowercased")
		require.True(t, tok.Swappable)
		require.Equal(t, "0", p.Reserves[i], "no reserves at discovery")
	}
	// No Extra at discovery, so the tracker's cold-start killed() backstop still runs once.
	require.Empty(t, p.Extra)
	require.Empty(t, p.StaticExtra)
	_ = entity.Pool(*p)
}

func TestPoolFactory_DecodePoolCreated_Invalid(t *testing.T) {
	f := NewPoolFactory(&Config{DexID: DexType})

	// Missing the indexed pool topic.
	_, err := f.DecodePoolCreated(ethtypes.Log{Topics: []common.Hash{partyStartedEventTopic}})
	require.ErrorIs(t, err, ErrInvalidEvent)
}
