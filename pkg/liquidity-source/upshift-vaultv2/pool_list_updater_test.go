package upshiftvaultv2

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const multicallAggregateABI = `[{"name":"aggregate","type":"function","stateMutability":"view",
"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`

// fakeMulticall answers multicall aggregate eth_calls: asset() and lpTokenAddress() per vault.
func fakeMulticall(t *testing.T, assets, lps map[common.Address]common.Address) *ethrpc.Client {
	t.Helper()
	mc, err := abi.JSON(strings.NewReader(multicallAggregateABI))
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage
			Params []json.RawMessage
		}
		require.NoError(t, json.Unmarshal(body, &req))
		var msg struct{ Input string }
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data, err := hexutil.Decode(msg.Input)
		require.NoError(t, err)
		args, err := mc.Methods["aggregate"].Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := abi.ConvertType(args[0], new([]struct {
			Target   common.Address
			CallData []byte
		})).(*[]struct {
			Target   common.Address
			CallData []byte
		})
		rets := make([][]byte, len(*calls))
		for i, c := range *calls {
			want := assets
			if string(c.CallData[:4]) == string(vaultABI.Methods[vaultMethodLP].ID) {
				want = lps
			} else {
				require.Equal(t, vaultABI.Methods[vaultMethodAsset].ID, c.CallData[:4])
			}
			addr := want[c.Target]
			rets[i] = common.LeftPadBytes(addr[:], 32)
		}
		out, err := mc.Methods["aggregate"].Outputs.Pack(big.NewInt(1), rets)
		require.NoError(t, err)
		resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(out)})
		_, _ = w.Write(resp)
	}))
	t.Cleanup(srv.Close)
	return ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
}

func TestGetNewPools(t *testing.T) {
	t.Parallel()
	// Mixed-case config addresses must come out lowercase (hexutil.Encode form).
	mixedVault := "0x36eDbF0C834591BFdfCaC0Ef9605528c75c406aA"
	vaultB := "0x8fFDcd8A96d293f45aA044d10b899F9D71897E8a"
	va, vb := common.HexToAddress(mixedVault), common.HexToAddress(vaultB)
	assetA, lpA := common.HexToAddress(ausd), common.HexToAddress(earnLP)
	assetB := common.HexToAddress("0x5d3A1Ff2b6BAb83b63cd9AD0787074081a52ef34")
	lpB := common.HexToAddress("0x54a98fF45a7DBDF30C54E32fD330d3eA582A5559")
	client := fakeMulticall(t,
		map[common.Address]common.Address{va: assetA, vb: assetB},
		map[common.Address]common.Address{va: lpA, vb: lpB})

	u := NewPoolsListUpdater(&Config{DexID: DexType, Vaults: []string{mixedVault, vaultB}}, client)
	pools, cursor, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, cursor)
	require.Len(t, pools, 2)

	for i, want := range []struct{ vault, asset, lp common.Address }{{va, assetA, lpA}, {vb, assetB, lpB}} {
		p := pools[i]
		assert.Equal(t, hexutil.Encode(want.vault[:]), p.Address)
		assert.Equal(t, strings.ToLower(p.Address), p.Address)
		assert.Equal(t, DexType, p.Exchange)
		assert.Equal(t, DexType, p.Type)
		require.Len(t, p.Tokens, 2)
		assert.Equal(t, hexutil.Encode(want.asset[:]), p.Tokens[0].Address)
		assert.Equal(t, hexutil.Encode(want.lp[:]), p.Tokens[1].Address)
		assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		// placeholder state must be loadable by the simulator
		_, err = NewPoolSimulator(p)
		require.NoError(t, err)
	}

	pools, _, err = u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, pools)
}
