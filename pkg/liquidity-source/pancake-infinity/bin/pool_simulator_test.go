package bin

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

	tests := []struct {
		name            string
		tokenIn         string
		tokenOut        string
		amountIn        string
		expectAmountOut string
		expectError     error
	}{
		{
			name:            "small amount in",
			tokenIn:         "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
			tokenOut:        "0x55d398326f99059ff775485246999027b3197955", // BUSD
			amountIn:        "1000000000000000",                           // 0.001 USDC
			expectAmountOut: "999801019898010",                            // ~0.00099 BUSD
			expectError:     nil,
		},
		{
			name:            "normal amount in",
			tokenIn:         "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
			tokenOut:        "0x55d398326f99059ff775485246999027b3197955", // BUSD
			amountIn:        "1000000000000000000",                        // 1 USDC
			expectAmountOut: "999801019898010198",                         // ~0.99 BUSD
			expectError:     nil,
		},
		{
			name:        "large amount in",
			tokenIn:     "0x55d398326f99059ff775485246999027b3197955", // BUSD
			tokenOut:    "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
			amountIn:    "1000000000000000000000000000",               // 1M USDC (USDC reserve is lower than 1M)
			expectError: ErrBinIDNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: utils.NewBig10(tc.amountIn),
				},
				TokenOut: tc.tokenOut,
			})

			if tc.expectError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, out.TokenAmountOut.Amount.String(), tc.expectAmountOut)
			}
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	var poolEnt entity.Pool

	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	testutil.TestCalcAmountIn(t, pSim)
}

func TestMergeSwap(t *testing.T) {
	var poolEnt entity.Pool
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	const (
		loop      = 20
		tokenIn   = "0x55d398326f99059ff775485246999027b3197955"
		tokenOut  = "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d"
		amountRaw = "1000000000000000000000" // 1_000 BUSD
	)

	amountIn := utils.NewBig10(amountRaw)
	amountInTotal := new(big.Int).Mul(amountIn, big.NewInt(int64(loop)))

	var amountOutSingle *big.Int
	t.Run("single large swap", func(t *testing.T) {
		pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
		assert.NoError(t, err)

		result, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountInTotal},
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)

		amountOutSingle = result.TokenAmountOut.Amount
	})

	var totalAmountOutMulti = big.NewInt(0)
	t.Run("multiple smaller swaps", func(t *testing.T) {
		pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
		assert.NoError(t, err)

		for range loop {
			result, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
				TokenOut:      tokenOut,
			})
			require.NoError(t, err)

			pSim.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: amountIn},
				TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: result.TokenAmountOut.Amount},
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			})

			totalAmountOutMulti.Add(totalAmountOutMulti, result.TokenAmountOut.Amount)
		}
	})

	t.Run("compare results", func(t *testing.T) {
		diff := new(big.Int).Sub(amountOutSingle, totalAmountOutMulti)
		ratio := new(big.Float).Quo(new(big.Float).SetInt(diff), new(big.Float).SetInt(amountOutSingle))

		maxAllowedDiff := big.NewFloat(0.005) // 0.05%
		assert.True(t, ratio.Cmp(maxAllowedDiff) < 0, "output mismatch too large: %.6f", ratio)
	})
}

func TestPancakeInfinityBin(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	// remember setup env before running test
	rpcEndpoint := os.Getenv("BSC_RPC_ENDPOINT")
	subgraphEndpoint := os.Getenv("PANCAKE_BIN_SUBGRAPH_ENDPOINT")

	config := &Config{
		ChainID:                int(valueobject.ChainIDBSC),
		DexID:                  "pancake-infinity-bin",
		SubgraphAPI:            subgraphEndpoint,
		UniversalRouterAddress: "0x5Dc88340E1c5c6366864Ee415d6034cadd1A9897",
		Permit2Address:         "0x31c2F6fcFf4F8759b3Bd5Bf0e1084A055615c768",
		Multicall3Address:      "0xcA11bde05977b3631167028862bE2a173976CA11",
		VaultAddress:           "0x238a358808379702088667322f80aC48bAd5e6c4",
		BinPoolManagerAddress:  "0xC697d2898e0D09264376196696c51D7aBbbAA4a9",
		NewPoolLimit:           200,
		AllowSubgraphError:     true,
	}

	rpcClient := ethrpc.New(rpcEndpoint).
		SetMulticallContract(common.HexToAddress(config.Multicall3Address))
	subgraphClient := graphqlpkg.NewClient(config.SubgraphAPI)

	quoter := shared.NewQuoter(shared.QuoterConfig{
		QuoterAddress: "0xC631f4B0Fc2Dd68AD45f74B2942628db117dD359",
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

			// Calculate input amount as 1% of reserve
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
