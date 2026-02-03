package stabull

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	multicall3Address = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

func TestPoolSimulator_ValidateAgainstViewOriginSwap2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name       string
		rpcEnv     string
		defaultRPC string
		chainID    uint
		factory    string
		fromBlock  uint64
	}{
		{
			name:       "Polygon",
			rpcEnv:     "STABULL_RPC_POLYGON",
			defaultRPC: "https://polygon-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			chainID:    137,
			factory:    "0x3c60234db40e6e5b57504e401b1cdc79d91faf89",
		},
		{
			name:       "Base",
			rpcEnv:     "STABULL_RPC_BASE",
			defaultRPC: "https://base-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			chainID:    8453,
			factory:    "0x86Ba17ebf8819f7fd32Cf1A43AbCaAe541A5BEbf",
		},
		{
			name:       "Ethereum",
			rpcEnv:     "STABULL_RPC_ETHEREUM",
			defaultRPC: "https://eth-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			chainID:    1,
			factory:    "0x2e9E34b5Af24b66F12721113C1C8FFcbB7Bc8051",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("chain    | pool         | case        | originSwap | simulated | deviation")
			rpcURL := os.Getenv(tt.rpcEnv)
			if rpcURL == "" {
				rpcURL = tt.defaultRPC
			}

			client := ethrpc.New(rpcURL)
			require.NotNil(t, client)
			client.SetMulticallContract(common.HexToAddress(multicall3Address))

			ctx := context.Background()
			updater := NewPoolsListUpdater(&Config{
				DexID:          "stabull-test",
				ChainID:        tt.chainID,
				FactoryAddress: tt.factory,
				NewPoolLimit:   100,
				HTTPConfig: HTTPConfig{
					BaseURL: "https://api.stabull.finance",
				},
			}, client)

			pools, _, err := updater.GetNewPools(ctx, nil)
			require.NoError(t, err)
			require.NotEmpty(t, pools)

			tracker, err := NewPoolTracker(&Config{DexID: "stabull-test"}, client)
			require.NoError(t, err)

			for _, poolInfo := range pools {
				var staticExtra StaticExtra
				require.NoError(t, json.Unmarshal([]byte(poolInfo.Extra), &staticExtra))

				reserves, updatedExtra, err := tracker.fetchPoolStateWithOraclesFromNode(
					ctx,
					poolInfo,
					staticExtra,
				)
				if err != nil {
					t.Logf("%s | %s | %s", tt.name, poolInfo.Address, "oracle fetch failed, retrying without oracles")
					reserves, updatedExtra, err = tracker.fetchPoolStateFromNode(ctx, poolInfo.Address)
					if err != nil {
						t.Logf("%s | %s | %s", tt.name, poolInfo.Address, "state fetch failed, skipping pool")
						continue
					}
				}
				if len(reserves) != 2 {
					t.Logf("%s | %s | %s", tt.name, poolInfo.Address, "unexpected reserve length, skipping pool")
					continue
				}

				extraBytes, err := json.Marshal(updatedExtra)
				require.NoError(t, err)

				if updatedExtra.OracleRates[0] == nil || updatedExtra.OracleRates[1] == nil {
					missing := "missing oracleRate"
					if updatedExtra.OracleRates[0] == nil && updatedExtra.OracleRates[1] == nil {
						missing = "missing baseOracle+quoteOracle"
					} else if updatedExtra.OracleRates[0] == nil {
						missing = "missing baseOracle"
					} else if updatedExtra.OracleRates[1] == nil {
						missing = "missing quoteOracle"
					}
					logErrorRow(t, tt.name, poolInfo.Address, "base-0.01", fmt.Errorf("%s", missing))
					logErrorRow(t, tt.name, poolInfo.Address, "quote-0.01", fmt.Errorf("%s", missing))
					logErrorRow(t, tt.name, poolInfo.Address, "base-1", fmt.Errorf("%s", missing))
					logErrorRow(t, tt.name, poolInfo.Address, "quote-1", fmt.Errorf("%s", missing))
					logErrorRow(t, tt.name, poolInfo.Address, "base-100", fmt.Errorf("%s", missing))
					logErrorRow(t, tt.name, poolInfo.Address, "quote-100", fmt.Errorf("%s", missing))
					continue
				}

				entityPool := entity.Pool{
					Address:  poolInfo.Address,
					Exchange: "stabull",
					Type:     DexType,
					Tokens: []*entity.PoolToken{
						{Address: poolInfo.Tokens[0].Address, Decimals: 18},
						{Address: poolInfo.Tokens[1].Address, Decimals: 18},
					},
					Reserves: []string{reserves[0].String(), reserves[1].String()},
					Extra:    string(extraBytes),
				}

				sim, err := NewPoolSimulator(entityPool)
				require.NoError(t, err)

				baseToken := poolInfo.Tokens[0]
				quoteToken := poolInfo.Tokens[1]

				poolLabel := fetchPoolSymbol(ctx, client, poolInfo.Address)
				if poolLabel == "" {
					poolLabel = poolInfo.Address
				}

				runCase(t, client, sim, poolInfo.Address, "base-1",
					baseToken.Address, quoteToken.Address,
					amountWithDecimals(1, int(baseToken.Decimals)),
					int(baseToken.Decimals), int(quoteToken.Decimals), tt.name, poolLabel,
				)
				runCase(t, client, sim, poolInfo.Address, "quote-1",
					quoteToken.Address, baseToken.Address,
					amountWithDecimals(1, int(quoteToken.Decimals)),
					int(quoteToken.Decimals), int(baseToken.Decimals), tt.name, poolLabel,
				)
				runCase(t, client, sim, poolInfo.Address, "base-0.01",
					baseToken.Address, quoteToken.Address,
					amountWithDecimalsString("0.01", int(baseToken.Decimals)),
					int(baseToken.Decimals), int(quoteToken.Decimals), tt.name, poolLabel,
				)
				runCase(t, client, sim, poolInfo.Address, "quote-0.01",
					quoteToken.Address, baseToken.Address,
					amountWithDecimalsString("0.01", int(quoteToken.Decimals)),
					int(quoteToken.Decimals), int(baseToken.Decimals), tt.name, poolLabel,
				)
				runCase(t, client, sim, poolInfo.Address, "base-100",
					baseToken.Address, quoteToken.Address,
					amountWithDecimals(100, int(baseToken.Decimals)),
					int(baseToken.Decimals), int(quoteToken.Decimals), tt.name, poolLabel,
				)
				runCase(t, client, sim, poolInfo.Address, "quote-100",
					quoteToken.Address, baseToken.Address,
					amountWithDecimals(100, int(quoteToken.Decimals)),
					int(quoteToken.Decimals), int(baseToken.Decimals), tt.name, poolLabel,
				)
			}
		})
	}
}

func runCase(
	t *testing.T,
	client *ethrpc.Client,
	sim *PoolSimulator,
	poolAddr string,
	name string,
	tokenIn string,
	tokenOut string,
	amountInRaw *big.Int,
	tokenInDecimals int,
	tokenOutDecimals int,
	chainLabel string,
	poolLabel string,
) {
	t.Helper()

	ctx := context.Background()
	var (
		contractOut  *big.Int
		originValue  string
		simValue     string
		deviation    string
		originRevert bool
	)
	swapRequest := client.NewRequest().SetContext(ctx)
	swapRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddr,
		Method: poolMethodViewOriginSwap,
		Params: []any{
			common.HexToAddress(tokenIn),
			common.HexToAddress(tokenOut),
			amountInRaw,
		},
	}, []any{&contractOut})
	resp, err := swapRequest.TryAggregate()
	if err != nil || len(resp.Result) == 0 || !resp.Result[0] || contractOut == nil {
		originValue = "revert"
		originRevert = true
	} else {
		originValue = formatTokenAmount(contractOut, tokenOutDecimals, 4)
	}

	amountIn18 := scaleAmount(amountInRaw, tokenInDecimals, 18)
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn18,
		},
		TokenOut: tokenOut,
	})
	if err != nil {
		simValue = "fail"
		deviation = err.Error()
	} else {
		simOut18 := result.TokenAmountOut.Amount
		simOut := scaleAmount(simOut18, 18, tokenOutDecimals)
		simValue = formatTokenAmount(simOut, tokenOutDecimals, 4)
		if !originRevert {
			deviation = deviationPercent(contractOut, simOut)
		}
	}
	if originValue == "" {
		originValue = "n/a"
	}
	if simValue == "" {
		simValue = "n/a"
	}

	t.Logf("%-8s | %-12s | %-10s | %-10s | %-9s | %s",
		chainLabel,
		poolLabel,
		name,
		originValue,
		simValue,
		deviation,
	)
}

func logErrorRow(t *testing.T, chainLabel string, poolLabel string, caseLabel string, err error) {
	t.Helper()
	t.Logf("%-8s | %-12s | %-10s | %-10s | %-9s | %s",
		chainLabel,
		poolLabel,
		caseLabel,
		"n/a",
		"n/a",
		err.Error(),
	)
}

func amountWithDecimals(units int, decimals int) *big.Int {
	if decimals < 0 {
		decimals = 0
	}
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	return new(big.Int).Mul(big.NewInt(int64(units)), base)
}

func amountWithDecimalsString(value string, decimals int) *big.Int {
	if decimals < 0 {
		decimals = 0
	}
	parts := strings.SplitN(value, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	for len(fracPart) < decimals {
		fracPart += "0"
	}
	if len(fracPart) > decimals {
		fracPart = fracPart[:decimals]
	}
	raw := intPart + fracPart
	if raw == "" {
		return bignumber.ZeroBI
	}
	val, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return bignumber.ZeroBI
	}
	return val
}

func scaleAmount(value *big.Int, fromDecimals int, toDecimals int) *big.Int {
	if value == nil {
		return bignumber.ZeroBI
	}
	if fromDecimals == toDecimals {
		return new(big.Int).Set(value)
	}
	if fromDecimals < toDecimals {
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(toDecimals-fromDecimals)), nil)
		return new(big.Int).Mul(value, multiplier)
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(fromDecimals-toDecimals)), nil)
	return new(big.Int).Div(value, divisor)
}

func deviationPercent(expected *big.Int, actual *big.Int) string {
	if expected == nil || expected.Cmp(bignumber.ZeroBI) == 0 {
		return "n/a"
	}
	diff := new(big.Int).Sub(actual, expected)
	absDiff := new(big.Int).Abs(diff)
	rat := new(big.Rat).SetFrac(absDiff, expected)
	rat.Mul(rat, big.NewRat(100, 1))
	sign := ""
	if diff.Sign() < 0 {
		sign = "-"
	}
	return fmt.Sprintf("%s%s%%", sign, rat.FloatString(4))
}

func formatTokenAmount(value *big.Int, decimals int, displayDecimals int) string {
	if value == nil {
		return "n/a"
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	rat := new(big.Rat).SetFrac(value, scale)
	return rat.FloatString(displayDecimals)
}

func fetchPoolSymbol(ctx context.Context, client *ethrpc.Client, poolAddr string) string {
	erc20SymbolABI, err := abi.JSON(bytes.NewReader([]byte(`[{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"}]`)))
	if err != nil {
		return ""
	}
	var symbol string
	req := client.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    erc20SymbolABI,
		Target: poolAddr,
		Method: "symbol",
		Params: []any{},
	}, []any{&symbol})
	if _, err := req.Call(); err != nil {
		return ""
	}
	return symbol
}
