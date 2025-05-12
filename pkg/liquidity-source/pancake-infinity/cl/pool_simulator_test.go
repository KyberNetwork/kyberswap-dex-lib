package cl

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	//go:embed sample_pool.json
	poolData string
	chainID  = 1
)

func TestCalcAmountOut(t *testing.T) {
	var poolEnt entity.Pool

	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
			Amount: utils.NewBig10("1000000000000000000"),
		},
		TokenOut: "0x55d398326f99059ff775485246999027b3197955",
	})
	assert.NoError(t, err)
	assert.Equal(t, utils.NewBig10("609097871894318314148"), got.TokenAmountOut.Amount)
}

func TestCalcAmountIn(t *testing.T) {
	var poolEnt entity.Pool

	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	testutil.TestCalcAmountIn(t, pSim)
}

func TestPancakeInfinityCL(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	// remember setup env before running test
	rpcEndpoint := os.Getenv("BSC_RPC_ENDPOINT")
	subgraphEndpoint := os.Getenv("PANCAKE_CL_SUBGRAPH_ENDPOINT")

	config := &Config{
		ChainID:                int(valueobject.ChainIDBSC),
		DexID:                  "pancake-infinity-cl",
		SubgraphAPI:            subgraphEndpoint,
		UniversalRouterAddress: "0x5Dc88340E1c5c6366864Ee415d6034cadd1A9897",
		Permit2Address:         "0x31c2F6fcFf4F8759b3Bd5Bf0e1084A055615c768",
		Multicall3Address:      "0xcA11bde05977b3631167028862bE2a173976CA11",
		VaultAddress:           "0x238a358808379702088667322f80aC48bAd5e6c4",
		CLPoolManagerAddress:   "0xa0FfB9c1CE1Fe56963B0321B32E7A0302114058b",
		NewPoolLimit:           200,
		AllowSubgraphError:     true,
	}

	rpcClient := ethrpc.New(rpcEndpoint).
		SetMulticallContract(common.HexToAddress(config.Multicall3Address))
	subgraphClient := graphqlpkg.NewClient(config.SubgraphAPI)

	quoter := shared.NewQuoter(shared.QuoterConfig{
		QuoterAddress: "0xd0737C9762912dD34c3271197E362Aa736Df0926",
	}, rpcClient)

	poolListUpdater := NewPoolListUpdater(config, rpcClient, subgraphClient)
	require.NotNil(t, poolListUpdater)

	poolTracker, err := NewPoolTracker(config, rpcClient, subgraphClient)
	require.NoError(t, err)
	require.NotNil(t, poolTracker)

	pools, metadata, err := poolListUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, pools)
	require.NotNil(t, metadata)

	numPoolsToTest := 100
	if len(pools) < numPoolsToTest {
		numPoolsToTest = len(pools)
	}

	var params pool.CalcAmountOutParams
	var poolMeta PoolMetaInfo
	var quoteParams shared.QuoteExactSingleParams
	var simulator *PoolSimulator
	var simulateRes *pool.CalcAmountOutResult
	var onchainRes shared.QuoteResult
	var simulateErr, onchainErr error
	var amountIn, tmp big.Int

	hundred := big.NewInt(100)
	tenThousand := big.NewInt(10000)

	// Function to check if the difference between two amounts is within 5 bps (0.05%)
	checkAmountOutDiff := func(amount1, amount2 *big.Int) bool {
		if amount1.Cmp(amount2) == 0 {
			return true
		}
		diff := new(big.Int).Abs(new(big.Int).Sub(amount1, amount2))
		maxDiff := new(big.Int).Div(new(big.Int).Mul(amount1, bignumber.Five), tenThousand)
		return diff.Cmp(maxDiff) <= 0
	}

	for _, p := range pickRandomPools(pools, numPoolsToTest) {
		require.NotNil(t, p)

		p, err = poolTracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.NotNil(t, p)

		simulator, err = NewPoolSimulator(p, valueobject.ChainID(config.ChainID))
		if simulator == nil {
			t.Logf("[%s] Failed to init Pool Simulator: %v", p.Address, err)
			continue
		}

		for i := range len(p.Tokens) {
			tokenIn := p.Tokens[i].Address
			tokenOut := p.Tokens[1-i].Address

			if len(p.Reserves[i]) < int(p.Tokens[i].Decimals) {
				continue
			}

			reserveIn := bignumber.NewBig(p.Reserves[i])

			// Calculate input amount as 1% of reserve and > 1 Gwei
			amountIn.Div(tmp.Mul(reserveIn, bignumber.One), hundred)
			if len(amountIn.String()) < int(p.Tokens[0].Decimals) {
				continue
			}

			params = pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tokenIn,
					Amount: &amountIn,
				},
				TokenOut: tokenOut,
			}

			// Calculate amount out offchain
			simulateRes, simulateErr = simulator.CalcAmountOut(params)

			poolMeta = simulator.GetMetaInfo(tokenIn, tokenOut).(PoolMetaInfo)

			quoteParams = shared.QuoteExactSingleParams{
				PoolKey: shared.PoolKey{
					Currency0: lo.Ternary(valueobject.IsWrappedNative(p.Tokens[0].Address, valueobject.ChainIDBSC),
						eth.AddressZero, common.HexToAddress(p.Tokens[0].Address)),
					Currency1: lo.Ternary(valueobject.IsWrappedNative(p.Tokens[1].Address, valueobject.ChainIDBSC),
						eth.AddressZero, common.HexToAddress(p.Tokens[1].Address)),
					Hooks:       poolMeta.HookAddress,
					PoolManager: poolMeta.PoolManager,
					Fee:         big.NewInt(int64(poolMeta.Fee)),
					Parameters:  common.HexToHash(poolMeta.Parameters),
				},
				ZeroForOne:  i == 0,
				ExactAmount: &amountIn,
				HookData:    poolMeta.HookData,
			}

			// Calculate amount out onchain
			onchainRes, onchainErr = quoter.QuoteExactInputSingle(context.Background(), quoteParams, p.BlockNumber)

			require.Equal(t, simulateErr == nil, onchainErr == nil, fmt.Sprintf("%s : output mismatch", p.Address), quoteParams, p.BlockNumber, simulateErr, onchainErr)

			if simulateRes != nil && onchainRes.AmountOut != nil {
				require.True(t, checkAmountOutDiff(simulateRes.TokenAmountOut.Amount, onchainRes.AmountOut),
					fmt.Sprintf("[%s] expected amount out is %v but got %v", p.Address, simulateRes.TokenAmountOut.Amount, onchainRes.AmountOut))
			}
		}
	}
}

func pickRandomPools(pools []entity.Pool, n int) []entity.Pool {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	indices := make(map[int]struct{}, n)
	selected := make([]entity.Pool, 0, n)
	for len(selected) < n {
		idx := r.Intn(len(pools))
		if _, exists := indices[idx]; !exists {
			indices[idx] = struct{}{}
			selected = append(selected, pools[idx])
		}
	}
	return selected
}
