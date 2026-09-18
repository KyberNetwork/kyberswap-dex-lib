package prmfun

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func fixtureMarkets() []GetMemeResult {
	markets := make([]GetMemeResult, 4)
	for i := range markets {
		m := &markets[i].Data
		m.Token = common.BigToAddress(big.NewInt(int64(100 + i)))
		m.Curve = common.BigToAddress(big.NewInt(int64(200 + i)))
		m.GraduationDesk = big.NewInt(int64(4200000 + i))
	}
	markets[1].Data.SubjectId = crypto.Keccak256Hash([]byte("premium.pair.PRM"))
	markets[2].Data.SubjectId = crypto.Keccak256Hash([]byte("existing-stock-subject"))
	return markets
}

func discoveryAnswer(markets []GetMemeResult, fail bool) func(common.Address, *abi.Method, []any) ([]any, bool) {
	pairs := []string{testDeskToken, prmToken, stockToken, testDeskToken}
	return func(target common.Address, method *abi.Method, args []any) ([]any, bool) {
		switch method.Name {
		case "memeCount":
			return []any{big.NewInt(int64(len(markets)))}, true
		case "memeTokens":
			return []any{markets[args[0].(*big.Int).Int64()].Data.Token}, true
		case "getMeme":
			for _, m := range markets {
				if m.Data.Token == args[0].(common.Address) {
					return []any{m.Data}, true
				}
			}
		}
		for i, m := range markets {
			if m.Data.Curve != target {
				continue
			}
			switch method.Name {
			case "deskToken":
				return []any{common.HexToAddress(pairs[i])}, !fail
			case "isNativeQuote":
				return []any{i == 0 || i == 3}, true
			case "phase":
				phase := PhaseTrading
				if i == 2 {
					phase = PhasePaused
				}
				if i == 3 {
					phase = PhaseGraduated
				}
				return []any{phase}, true
			}
		}
		return nil, false
	}
}

func TestDiscoveryAllPairsBackfillAndPaused(t *testing.T) {
	markets := fixtureMarkets()
	client := mockClient(t, discoveryAnswer(markets, false))
	lister := NewPoolsListUpdater(&Config{DexId: DexType, ChainId: 4663, FactoryAddress: factoryAddress, RouterAddress: routerAddress, NewPoolLimit: 3}, client)
	pools, metadata, err := lister.GetNewPools(context.Background(), []byte(`{"offset":4}`))
	require.NoError(t, err)
	require.Len(t, pools, 3)
	for i, p := range pools {
		require.Equal(t, []string{testDeskToken, prmToken, stockToken}[i], p.Tokens[0].Address)
		require.Equal(t, []string{"0", "0"}, []string(p.Reserves))
		var extra StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &extra))
		require.Equal(t, i == 0, extra.IsNativeQuote)
		require.Equal(t, markets[i].Data.GraduationDesk.String(), extra.GraduationDesk)
	}
	var m PoolsListUpdaterMetadata
	require.NoError(t, json.Unmarshal(metadata, &m))
	require.Equal(t, PoolsListUpdaterMetadata{Offset: 3, Version: 2}, m)
	// The paused stock pair was admitted; the graduated final curve is skipped.
	next, metadata, err := lister.GetNewPools(context.Background(), metadata)
	require.NoError(t, err)
	require.Empty(t, next)
	require.NoError(t, json.Unmarshal(metadata, &m))
	require.Equal(t, 4, m.Offset)
}

func TestDiscoveryFailedBatchRetainsCursor(t *testing.T) {
	client := mockClient(t, discoveryAnswer(fixtureMarkets(), true))
	lister := NewPoolsListUpdater(&Config{DexId: DexType, ChainId: 4663, FactoryAddress: factoryAddress, RouterAddress: routerAddress}, client)
	before := []byte(`{"offset":0,"version":2}`)
	pools, after, err := lister.GetNewPools(context.Background(), before)
	require.Error(t, err)
	require.Empty(t, pools)
	require.Equal(t, before, after)
}

func TestDiscoveryRejectsMalformedCursor(t *testing.T) {
	for _, data := range []string{`{`, `{"offset":-1,"version":2}`} {
		_, err := discoveryMetadata([]byte(data))
		require.Error(t, err)
	}
}
