package inverse

import (
	"context"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// Read-only opt-in verification of the public launch's two mined buys. Historical
// RPC access is required. The exported vectors can be tested without an RPC.
func TestRobinhoodDeployment(t *testing.T) {
	path := os.Getenv("INVERSE_LIVE_MANIFEST")
	if path == "" {
		t.Skip("set INVERSE_LIVE_MANIFEST and INVERSE_LIVE_RPC for read-only live verification")
	}
	var launch struct {
		Stage string `json:"stage"`
		Plan  struct {
			Hook      common.Address `json:"hook"`
			Token     common.Address `json:"token"`
			Wallet    common.Address `json:"wallet"`
			PoolID    common.Hash    `json:"poolId"`
			BuyAmount uint64         `json:"buyAmount"`
		} `json:"plan"`
		Transactions []common.Hash `json:"transactions"`
	}
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &launch))
	require.Equal(t, "deployed_and_two_buys_verified", launch.Stage)
	require.GreaterOrEqual(t, len(launch.Transactions), 2)
	rpc := os.Getenv("INVERSE_LIVE_RPC")
	require.NotEmpty(t, rpc)
	client, err := ethclient.Dial(rpc)
	require.NoError(t, err)
	defer client.Close()
	ctx := context.Background()
	chain, err := client.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(4663), chain.Int64())
	trackerRPC := ethrpc.New(rpc).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	hook, token := launch.Plan.Hook, launch.Plan.Token
	token0, token1 := token, quoteAddress
	inverse0 := token.Cmp(quoteAddress) < 0
	if !inverse0 {
		token0, token1 = token1, token0
	}
	static, err := json.Marshal(uniswapv4.StaticExtra{Fee: 3000, TickSpacing: 60, HooksAddress: hook})
	require.NoError(t, err)
	track := func(block *big.Int) Extra {
		p := &entity.Pool{Address: launch.Plan.PoolID.Hex(), Tokens: []*entity.PoolToken{{Address: hexutil.Encode(token0[:]), Decimals: 18}, {Address: hexutil.Encode(token1[:]), Decimals: 18}}, StaticExtra: string(static)}
		param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: 4663}, RpcClient: trackerRPC, Pool: p, BlockNumber: block}
		h, registered := uniswapv4.GetHook(hook, param)
		require.True(t, registered)
		require.IsType(t, &Hook{}, h)
		_, e := h.Track(ctx, param)
		require.NoError(t, e)
		require.Equal(t, block.Uint64(), p.BlockNumber)
		return h.(*Hook).State
	}
	transfer := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	var fixtures []vector
	for i, hash := range launch.Transactions[len(launch.Transactions)-2:] {
		receipt, e := client.TransactionReceipt(ctx, hash)
		require.NoError(t, e)
		require.Equal(t, uint64(1), receipt.Status)
		before := track(new(big.Int).Sub(receipt.BlockNumber, n(1)))
		after := track(receipt.BlockNumber)
		// Local transitions retain the refresh block; the next refresh advances it.
		after.BlockNumber = before.BlockNumber
		var output *big.Int
		for _, log := range receipt.Logs {
			if log.Address == token && len(log.Topics) == 3 && log.Topics[0] == transfer &&
				common.BytesToAddress(log.Topics[1].Bytes()) == managerAddress &&
				common.BytesToAddress(log.Topics[2].Bytes()) == launch.Plan.Wallet {
				require.Nil(t, output, "one complete output take is required")
				output = new(big.Int).SetBytes(log.Data)
			}
		}
		require.NotNil(t, output)
		input := new(big.Int).SetUint64(launch.Plan.BuyAmount)
		quoted, next, e := quote(before, !inverse0, input)
		require.NoError(t, e)
		require.Equal(t, output.String(), quoted.String())
		require.Equal(t, after, next, "all modeled state fields must match mined execution")
		fixtures = append(fixtures, vector{Name: "robinhood-buy-" + strconv.Itoa(i+1), Before: before, After: after, ZeroForOne: !inverse0, Input: *u(input), Output: *u(output), Success: true})
		t.Logf("block %d tx %s: output=%s; all modeled fields match", receipt.BlockNumber.Uint64(), hash.Hex(), output)
	}
	if out := os.Getenv("INVERSE_LIVE_FIXTURE_OUT"); out != "" {
		raw, err = json.MarshalIndent(&fixtures, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(out, append(raw, '\n'), 0644))
	}
}
