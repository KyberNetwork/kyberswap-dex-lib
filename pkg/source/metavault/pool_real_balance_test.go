package metavault

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// healthy vault of 3 tokens (A0, A2, A10 of the data in TestPoolSimulator_CalcAmountOut), where the real balances
// and the vault accounting of each token are given by realBalances / tokenBalances (a JSON object body).
func newRealBalanceTestVault(t *testing.T, realBalances, tokenBalances string) *Vault {
	t.Helper()
	extra := fmt.Sprintf(`{"vault":{"hasDynamicFees":true,"includeAmmPrice":false,"isSwapEnabled":true,"stableSwapFeeBasisPoints":25,"stableTaxBasisPoints":5,"swapFeeBasisPoints":30,"taxBasisPoints":50,"totalTokenWeights":100000,
"whitelistedTokens":["A0","A2","A10"],
"poolAmounts":{"A0":351500182590784658632430,"A2":582500526946365638607,"A10":54956975689863757124184},
"bufferAmounts":{"A0":1,"A2":1,"A10":1},
"reservedAmounts":{"A0":10076951923665922051838,"A2":177431951598853399981,"A10":0},
"tokenDecimals":{"A0":18,"A2":18,"A10":18},
"stableTokens":{"A0":false,"A2":false,"A10":false},
"usdmAmounts":{"A0":253813004375598635984875,"A2":1093840701705446620699791,"A10":39966351629451563348533},
"maxUsdmAmounts":{"A0":400000000000000000000000,"A2":1400000000000000000000000,"A10":40000000000000000000000},
"tokenWeights":{"A0":8000,"A2":28000,"A10":1000},
"tokenBalances":{%s},"realBalances":{%s},
"priceFeed":{"isSecondaryPriceEnabled":true,"maxStrictPriceDeviation":15000000000000000000000000000,"priceSampleSpace":1,
"priceDecimals":{"A0":8,"A2":8,"A10":8},"spreadBasisPoints":{"A0":8,"A2":0,"A10":17},"adjustmentBasisPoints":{"A0":0,"A2":0,"A10":0},
"strictStableTokens":{"A0":false,"A2":false,"A10":false},"isAdjustmentAdditive":{"A0":false,"A2":false,"A10":false},
"secondaryPriceFeed":{"disableFastPriceVoteCount":0,"isSpreadEnabled":false,"lastUpdatedAt":%d,"maxDeviationBasisPoints":100,"minAuthorizations":1,"priceDuration":300,"maxPriceUpdateDelay":3600,"spreadBasisPointsIfChainError":500,"spreadBasisPointsIfInactive":50,
"prices":{"A0":690650000000000000000000000000,"A2":1964120000000000000000000000000000,"A10":0},
"priceData":{"A0":{"refPrice":0,"refTime":0,"cumulativeRefDelta":0,"cumulativeFastDelta":0},"A2":{"refPrice":0,"refTime":0,"cumulativeRefDelta":0,"cumulativeFastDelta":0},"A10":{"refPrice":0,"refTime":0,"cumulativeRefDelta":0,"cumulativeFastDelta":0}},
"maxCumulativeDeltaDiffs":{"A0":50000,"A2":50000,"A10":0}},
"secondaryPriceFeedVersion":2,
"priceFeeds":{"A0":{"roundId":1,"answer":69050000,"answers":{"1":69050000}},"A2":{"roundId":1,"answer":196345780000,"answers":{"1":196345780000}},"A10":{"roundId":1,"answer":74353248,"answers":{"1":74353248}}}},
"usdm":{"address":"0x533403a3346cA31D67c380917ffaF185c24e7333","totalSupply":3389201190535341442726377}}}`,
		tokenBalances, realBalances, time.Now().Unix())

	var e Extra
	require.NoError(t, json.Unmarshal([]byte(extra), &e))
	return e.Vault
}

const (
	// poolAmounts of the fixture, the healthy state has the vault holding exactly that
	poolA0  = "351500182590784658632430"
	poolA2  = "582500526946365638607"
	poolA10 = "54956975689863757124184"

	balancesHealthy = `"A0":` + poolA0 + `,"A2":` + poolA2 + `,"A10":` + poolA10
	// A0 is 1 wei below its pool amount while the accounting is synced to the real balance: only
	// poolAmounts + credited <= balanceOf (Vault._increasePoolAmount, code 49) catches it
	balancesA0BelowPool = `"A0":351500182590784658632429,"A2":` + poolA2 + `,"A10":` + poolA10
	// polygon drained vault: the vault holds none of A0 and A2 (accounting synced or not)
	balancesA0A2Zero = `"A0":0,"A2":0,"A10":` + poolA10

	sent = 1_000_000_000_000_000 // 1e15, the amount sent to the vault in the tests
)

func bal(a0, a2, a10 string) string {
	return fmt.Sprintf(`"A0":%s,"A2":%s,"A10":%s`, a0, a2, a10)
}

func bigStr(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

func newRealBalanceTestSimulator(t *testing.T, realBalances, tokenBalances string) *PoolSimulator {
	t.Helper()
	extraVault := newRealBalanceTestVault(t, realBalances, tokenBalances)
	extraBytes, err := json.Marshal(Extra{Vault: extraVault})
	require.NoError(t, err)
	p, err := NewPoolSimulator(entity.Pool{
		Tokens:   []*entity.PoolToken{{Address: "A0"}, {Address: "A2"}, {Address: "A10"}},
		Reserves: entity.PoolReserves{"1", "1", "1"},
		Extra:    string(extraBytes),
	})
	require.NoError(t, err)
	return p
}

func calcAmountOutOf(p *PoolSimulator, in, out string, amount *big.Int) (*pool.CalcAmountOutResult, error) {
	return p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount},
		TokenOut:      out,
	})
}

func calcAmountOut(p *PoolSimulator, in, out string) (*pool.CalcAmountOutResult, error) {
	return calcAmountOutOf(p, in, out, big.NewInt(sent))
}

func swapAndUpdate(t *testing.T, p *PoolSimulator, in, out string, amount *big.Int) *pool.CalcAmountOutResult {
	t.Helper()
	res, err := calcAmountOutOf(p, in, out, amount)
	require.NoError(t, err)
	p.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: in, Amount: amount},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})
	return res
}

func TestPoolSimulator_RealBalance(t *testing.T) {
	t.Parallel()

	// the quote of a healthy vault (real == tokenBalances == poolAmounts) for an amount sent
	healthyQuote := func(t *testing.T, in, out string, amount *big.Int) *big.Int {
		res, err := calcAmountOutOf(newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy), in, out, amount)
		require.NoError(t, err)
		return res.TokenAmountOut.Amount
	}

	t.Run("healthy vault is quoted as before", func(t *testing.T) {
		p := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
		res, err := calcAmountOut(p, "A0", "A2")
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(350403344400), res.TokenAmountOut.Amount) // same as TestPoolSimulator_CalcAmountOut
	})

	// _transferIn credits balanceOf - tokenBalances: a donation is credited to the swapper
	t.Run("surplus of the real balance over the accounting is credited", func(t *testing.T) {
		donation := big.NewInt(500_000_000_000_000)
		surplusA0 := new(big.Int).Add(bigStr(poolA0), donation).String()
		p := newRealBalanceTestSimulator(t, bal(surplusA0, poolA2, poolA10), balancesHealthy)
		res, err := calcAmountOut(p, "A0", "A2")
		require.NoError(t, err)
		want := healthyQuote(t, "A0", "A2", new(big.Int).Add(big.NewInt(sent), donation))
		assert.Equal(t, want, res.TokenAmountOut.Amount)
		assert.Equal(t, 1, res.TokenAmountOut.Amount.Cmp(healthyQuote(t, "A0", "A2", big.NewInt(sent))))
	})

	t.Run("small deficit is credited less, no revert", func(t *testing.T) {
		deficit := big.NewInt(400_000_000_000_000)
		deficitA0 := new(big.Int).Sub(bigStr(poolA0), deficit).String()
		// the accounting still shows poolA0 (real < tokenBalances), and the pool amount too: code 49 holds at equality
		p := newRealBalanceTestSimulator(t, bal(deficitA0, poolA2, poolA10), balancesHealthy)
		res, err := calcAmountOut(p, "A0", "A2")
		require.NoError(t, err)
		want := healthyQuote(t, "A0", "A2", new(big.Int).Sub(big.NewInt(sent), deficit))
		assert.Equal(t, want, res.TokenAmountOut.Amount)
		assert.Equal(t, -1, res.TokenAmountOut.Amount.Cmp(healthyQuote(t, "A0", "A2", big.NewInt(sent))))
	})

	t.Run("deficit not below the amount sent: nothing is credited", func(t *testing.T) {
		// deficit == sent: credited 0 (code 27)
		eq := new(big.Int).Sub(bigStr(poolA0), big.NewInt(sent)).String()
		p := newRealBalanceTestSimulator(t, bal(eq, poolA2, poolA10), balancesHealthy)
		_, err := calcAmountOut(p, "A0", "A2")
		assert.ErrorIs(t, err, ErrVaultInvalidAmountIn)
		// deficit > sent: transferIn underflows
		above := new(big.Int).Sub(bigStr(poolA0), big.NewInt(sent+1)).String()
		p = newRealBalanceTestSimulator(t, bal(above, poolA2, poolA10), balancesHealthy)
		_, err = calcAmountOut(p, "A0", "A2")
		assert.ErrorIs(t, err, ErrVaultSubOverflow)
		// other pairs are not affected
		_, err = calcAmountOut(p, "A2", "A10")
		assert.NoError(t, err)
	})

	// poolAmounts[A0] > real balance although tokenBalances == real balance: _increasePoolAmount reverts (code 49)
	t.Run("code 49: pool amount above the balance", func(t *testing.T) {
		p := newRealBalanceTestSimulator(t, balancesA0BelowPool, balancesA0BelowPool)
		_, err := calcAmountOut(p, "A0", "A2")
		assert.ErrorIs(t, err, ErrVaultPoolAmountExceedsBalance)
		// as tokenOut it is fine: the vault holds enough to pay the amount out
		_, err = calcAmountOut(p, "A2", "A0")
		assert.NoError(t, err)
	})

	// transferOut of amountOutAfterFees: the boundary is exact
	t.Run("tokenOut balance against the amount out after fees", func(t *testing.T) {
		out := big.NewInt(350403344400)
		for _, tc := range []struct {
			real *big.Int
			ok   bool
		}{{out, true}, {new(big.Int).Sub(out, big.NewInt(1)), false}, {big.NewInt(0), false}} {
			p := newRealBalanceTestSimulator(t, bal(poolA0, tc.real.String(), poolA10), balancesHealthy)
			res, err := calcAmountOut(p, "A0", "A2")
			if tc.ok {
				require.NoError(t, err)
				assert.Equal(t, out, res.TokenAmountOut.Amount)
			} else {
				assert.ErrorIs(t, err, ErrInsufficientRealBalance)
			}
		}
	})

	t.Run("unknown balances (extra saved before the fields existed) keep the legacy behaviour", func(t *testing.T) {
		// no balances at all, or no accounting, or no real balance of tokenIn: the amount sent is credited and the
		// quote is the healthy one
		for name, p := range map[string]*PoolSimulator{
			"no balances":            newRealBalanceTestSimulator(t, ``, ``),
			"no real balances":       newRealBalanceTestSimulator(t, ``, balancesA0A2Zero),
			"no accounting of A0":    newRealBalanceTestSimulator(t, balancesHealthy, `"A2":0`),
			"unread real balance A0": newRealBalanceTestSimulator(t, `"A0":null,"A2":`+poolA2, balancesA0A2Zero),
		} {
			res, err := calcAmountOut(p, "A0", "A2")
			require.NoError(t, err, name)
			assert.Equal(t, big.NewInt(350403344400), res.TokenAmountOut.Amount, name)
		}
		// and UpdateBalance does not invent balances
		p := newRealBalanceTestSimulator(t, ``, ``)
		swapAndUpdate(t, p, "A0", "A2", big.NewInt(sent))
		assert.Empty(t, p.vault.RealBalances)
		assert.Empty(t, p.vault.TokenBalances)
	})

	// polygon metavault 0x32848e2d3aecfa7364595609fb050a301050a6b4: the vault holds none of most tokens while its
	// accounting still shows funds
	t.Run("drained vault is not quoted in either direction", func(t *testing.T) {
		for name, accounting := range map[string]string{
			"accounting still shows funds": balancesHealthy,
			"accounting synced to zero":    balancesA0A2Zero,
		} {
			p := newRealBalanceTestSimulator(t, balancesA0A2Zero, accounting)
			_, err := calcAmountOut(p, "A0", "A10") // tokenIn drained
			assert.Error(t, err, name)
			_, err = calcAmountOut(p, "A10", "A0") // tokenOut drained
			assert.ErrorIs(t, err, ErrInsufficientRealBalance, name)
			_, err = calcAmountOut(p, "A2", "A0")
			assert.Error(t, err, name)
		}
		p := newRealBalanceTestSimulator(t, balancesA0A2Zero, balancesHealthy)
		_, err := calcAmountOut(p, "A0", "A10")
		assert.ErrorIs(t, err, ErrVaultSubOverflow)
		p = newRealBalanceTestSimulator(t, balancesA0A2Zero, balancesA0A2Zero)
		_, err = calcAmountOut(p, "A0", "A10")
		assert.ErrorIs(t, err, ErrVaultPoolAmountExceedsBalance)
	})

	t.Run("recovery: same vault with the balances back quotes again", func(t *testing.T) {
		drained := newRealBalanceTestSimulator(t, balancesA0A2Zero, balancesHealthy)
		_, err := calcAmountOut(drained, "A0", "A10")
		require.Error(t, err)

		recovered := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
		_, err = calcAmountOut(recovered, "A0", "A10")
		assert.NoError(t, err)
	})
}

func TestPoolSimulator_UpdateBalanceChain(t *testing.T) {
	t.Parallel()

	t.Run("healthy: balances follow the transfers and stay consistent", func(t *testing.T) {
		p := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
		cloned := p.CloneState().(*PoolSimulator)

		res := swapAndUpdate(t, p, "A0", "A2", big.NewInt(sent))
		a0, a2 := bigStr(poolA0), bigStr(poolA2)
		wantA0 := new(big.Int).Add(a0, big.NewInt(sent))
		wantA2 := new(big.Int).Sub(a2, res.TokenAmountOut.Amount)
		assert.Equal(t, wantA0, p.vault.RealBalances["A0"])
		assert.Equal(t, wantA0, p.vault.TokenBalances["A0"])
		assert.Equal(t, wantA2, p.vault.RealBalances["A2"])
		assert.Equal(t, wantA2, p.vault.TokenBalances["A2"])
		assert.Equal(t, new(big.Int).Add(a0, big.NewInt(sent)), p.vault.PoolAmounts["A0"])
		// the vault pays amountOut = amountOutAfterFees + fee out of the pool, only amountOutAfterFees leaves the vault
		assert.Equal(t, new(big.Int).Sub(a2, new(big.Int).Add(res.TokenAmountOut.Amount, res.Fee.Amount)),
			p.vault.PoolAmounts["A2"])

		// the clone did not move
		assert.Equal(t, a0, cloned.vault.RealBalances["A0"])
		assert.Equal(t, a0, cloned.vault.TokenBalances["A0"])
		assert.Equal(t, a2, cloned.vault.RealBalances["A2"])
		assert.Equal(t, a0, cloned.vault.PoolAmounts["A0"])

		// second hop out of the token just received, then back: still consistent and quotable
		swapAndUpdate(t, p, "A2", "A0", big.NewInt(sent))
		for _, token := range []string{"A0", "A2", "A10"} {
			assert.Equal(t, p.vault.RealBalances[token], p.vault.TokenBalances[token], token)
		}
		_, err := calcAmountOut(p, "A0", "A10")
		assert.NoError(t, err)
	})

	// a donation (or a deficit) is absorbed by the first swap through the token only: afterwards the accounting is
	// synced to the real balance, so the second hop is credited exactly what it sent
	for name, delta := range map[string]int64{"surplus": 500_000_000_000_000, "deficit": -400_000_000_000_000} {
		t.Run(name+" is absorbed by the first hop only", func(t *testing.T) {
			realA0 := new(big.Int).Add(bigStr(poolA0), big.NewInt(delta)).String()
			p := newRealBalanceTestSimulator(t, bal(realA0, poolA2, poolA10), balancesHealthy)
			// reference: a healthy vault that is sent the credited amount of the first hop
			ref := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)

			swapAndUpdate(t, p, "A0", "A2", big.NewInt(sent))
			swapAndUpdate(t, ref, "A0", "A2", big.NewInt(sent+delta))
			assert.Equal(t, ref.vault.PoolAmounts, p.vault.PoolAmounts)
			assert.Equal(t, ref.vault.USDMAmounts, p.vault.USDMAmounts)
			assert.Equal(t, ref.vault.RealBalances["A2"], p.vault.RealBalances["A2"])
			// real A0 differs from the reference by the delta, and the accounting follows it
			assert.Equal(t, new(big.Int).Add(bigStr(realA0), big.NewInt(sent)), p.vault.RealBalances["A0"])
			assert.Equal(t, p.vault.RealBalances["A0"], p.vault.TokenBalances["A0"])
			credited, err := p.vault.CreditedAmountIn("A0", big.NewInt(sent))
			require.NoError(t, err)
			assert.Equal(t, big.NewInt(sent), credited)

			// so the second hop quotes the same as the reference does, from the same pool and usdm amounts
			got, err := calcAmountOut(p, "A0", "A10")
			require.NoError(t, err)
			want, err := calcAmountOut(ref, "A0", "A10")
			require.NoError(t, err)
			assert.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
		})
	}
}

func TestVaultReader_ReadTokensData_Balances(t *testing.T) {
	t.Parallel()
	const vaultAddr = "0x32848e2d3aecfa7364595609fb050a301050a6b4"
	tokens := []string{"0x1111111111111111111111111111111111111111", "0x2222222222222222222222222222222222222222"}
	accounting := map[string]*big.Int{tokens[0]: big.NewInt(500), tokens[1]: big.NewInt(700)}
	// the second token reverts on balanceOf
	realBal := map[string]*big.Int{tokens[0]: big.NewInt(0)}

	stub := testutil.NewMulticallStub(t, func(target common.Address, data []byte) (bool, []byte) {
		if target != common.HexToAddress(vaultAddr) {
			bal, ok := realBal[target.Hex()]
			if !ok {
				bal, ok = realBal[fmt.Sprintf("0x%x", target.Bytes())]
			}
			if !ok {
				return false, nil
			}
			ret, err := erc20ABI.Methods[ERC20MethodBalanceOf].Outputs.Pack(bal)
			require.NoError(t, err)
			return true, ret
		}
		method, err := vaultABI.MethodById(data[:4])
		require.NoError(t, err)
		args, err := method.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		var value any = big.NewInt(1)
		switch method.Name {
		case VaultMethodStableTokens:
			value = false
		case VaultMethodTokenBalances:
			value = accounting[fmt.Sprintf("0x%x", args[0].(common.Address).Bytes())]
		}
		ret, err := method.Outputs.Pack(value)
		require.NoError(t, err)
		return true, ret
	}).At(100, 1_700_000_000)

	vault := NewVault()
	vault.WhitelistedTokens = tokens
	require.NoError(t, NewVaultReader(stub.Client()).readTokensData(context.Background(), vaultAddr, vault))

	assert.Equal(t, accounting[tokens[0]], vault.TokenBalances[tokens[0]])
	assert.Equal(t, accounting[tokens[1]], vault.TokenBalances[tokens[1]])
	require.NotNil(t, vault.RealBalances[tokens[0]])
	assert.Zero(t, vault.RealBalances[tokens[0]].Sign())
	assert.Nil(t, vault.RealBalances[tokens[1]]) // balanceOf failed: unknown, not 0
	assert.Equal(t, uint64(100), vault.BlockNumber.Uint64())
	// one multicall: 10 calls per token
	require.Len(t, stub.Targets(), 1)
	assert.Len(t, stub.Targets()[0], 10*len(tokens))
}

func TestPoolSimulator_MissingPriceFeed(t *testing.T) {
	t.Parallel()
	p := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
	delete(p.vault.PriceFeed.PriceFeeds, "A2")

	_, err := calcAmountOut(p, "A0", "A2")
	assert.ErrorIs(t, err, ErrVaultPriceFeedInvalidPriceFeed)
	_, err = calcAmountOut(p, "A2", "A0")
	assert.ErrorIs(t, err, ErrVaultPriceFeedInvalidPriceFeed)
	_, err = calcAmountOut(p, "A0", "A10")
	assert.NoError(t, err)
}

func TestVaultScanner_GetPriceFeeds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	addrA, addrB := common.HexToAddress("0xa"), common.HexToAddress("0xb")
	feeds := map[string]common.Address{"A": addrA, "B": addrB}

	newScanner := func(ctrl *gomock.Controller, errB error) *VaultScanner {
		reader := NewMockIPriceFeedReader(ctrl)
		reader.EXPECT().Read(ctx, addrA.String(), 1).Return(NewPriceFeed(), nil).AnyTimes()
		reader.EXPECT().Read(ctx, addrB.String(), 1).Return(nil, errB).AnyTimes()
		return &VaultScanner{priceFeedReader: reader, log: logger.WithFields(logger.Fields{})}
	}

	t.Run("a feed that reverts is skipped, the other tokens are kept", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		got, err := newScanner(ctrl, ErrVaultPriceFeedLatestRoundDataReverted).getPriceFeeds(ctx, feeds, big.NewInt(1))
		require.NoError(t, err)
		assert.Contains(t, got, "A")
		assert.NotContains(t, got, "B")
	})

	t.Run("any other error still fails the scan", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		_, err := newScanner(ctrl, errors.New("rpc down")).getPriceFeeds(ctx, feeds, big.NewInt(1))
		assert.Error(t, err)
	})
}
