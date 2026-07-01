package caliberprop

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type liveChain struct {
	name       string
	rpcEnv     string
	rpcDefault string
	cfg        Config
}

var liveChains = []liveChain{
	{
		name:       "optimism",
		rpcEnv:     "OPTIMISM_RPC_URL",
		rpcDefault: "https://optimism.kyberengineering.io",
		cfg: Config{
			DexID:   DexType,
			Address: "0x60a8fA0eB9eDBF97a7487f7163C793768385Adc4",
		},
	},
	{
		name:       "base",
		rpcEnv:     "BASE_RPC_URL",
		rpcDefault: "https://base.kyberengineering.io",
		cfg: Config{
			DexID:   DexType,
			Address: "0xf639CF213b63F7E77D699FF686d591C0Ba55Fc63",
		},
	},
	{
		name:       "linea",
		rpcEnv:     "LINEA_RPC_URL",
		rpcDefault: "https://linea.kyberengineering.io",
		cfg: Config{
			DexID:   DexType,
			Address: "0xf639CF213b63F7E77D699FF686d591C0Ba55Fc63",
		},
	},
}

const multicallAddr = "0xcA11bde05977b3631167028862bE2a173976CA11"

func TestCaliberLiveQuoteParity(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("network test")
	}

	for _, lc := range liveChains {
		t.Run(lc.name, func(t *testing.T) {
			rpcURL := os.Getenv(lc.rpcEnv)
			if rpcURL == "" {
				rpcURL = lc.rpcDefault
			}
			client := ethrpc.New(rpcURL).
				SetMulticallContract(common.HexToAddress(multicallAddr))

			lister := NewPoolsListUpdater(&lc.cfg, client)
			pools, _, err := lister.GetNewPools(context.Background(), nil)
			require.NoError(t, err)

			tracker, err := NewPoolTracker(&lc.cfg, client)
			require.NoError(t, err)

			xors := common.HexToAddress(lc.cfg.Address)
			for _, ep := range pools {
				p, err := tracker.GetNewPoolState(context.Background(), ep, pool.GetNewPoolStateParams{})
				require.NoError(t, err)
				t.Logf("[%s] pair %s reserves=%v block=%d extra=%s",
					lc.name, p.Address, p.Reserves, p.BlockNumber, p.Extra)

				sim, err := NewPoolSimulator(p)
				require.NoError(t, err)

				token0 := common.HexToAddress(p.Tokens[0].Address)
				token1 := common.HexToAddress(p.Tokens[1].Address)
				dec0 := p.Tokens[0].Decimals
				dec1 := p.Tokens[1].Decimals
				pairID := common.HexToHash(p.Address)
				for i, xor := range xors {
					pairID[i] ^= xor
				}

				checkDirection(t, client, sim, p.BlockNumber, lc.cfg.Address, pairID,
					token0, token1, dec0, []int64{1, 5, 20, 100})
				checkDirection(t, client, sim, p.BlockNumber, lc.cfg.Address, pairID,
					token1, token0, dec1, []int64{1000, 5000, 20000, 100000})
			}
		})
	}
}

func checkDirection(
	t *testing.T,
	client *ethrpc.Client,
	sim *PoolSimulator,
	blockNumber uint64,
	contract string,
	pairID common.Hash,
	tokenIn, tokenOut common.Address,
	decIn uint8,
	wholeAmounts []int64,
) {
	t.Helper()
	for _, whole := range wholeAmounts {
		amt := new(big.Int).Mul(big.NewInt(whole), bignumber.TenPowInt(int(decIn)))

		var quoterOut *big.Int
		req := client.NewRequest().SetContext(context.Background())
		if blockNumber > 0 {
			req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
		}
		req.AddCall(&ethrpc.Call{
			ABI:    caliberABI,
			Target: contract,
			Method: methodQuote,
			Params: []any{pairID, tokenIn, tokenOut, amt},
		}, []any{&quoterOut})
		_, qErr := req.Call()

		tokenInLower := strings.ToLower(tokenIn.Hex())
		tokenOutLower := strings.ToLower(tokenOut.Hex())
		simRes, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenInLower, Amount: amt},
			TokenOut:      tokenOutLower,
		})

		label := fmt.Sprintf("%s->%s amt=%s", tokenInLower, tokenOutLower, amt)

		if qErr != nil || quoterOut == nil || quoterOut.Sign() == 0 {
			if simErr == nil && simRes != nil && simRes.TokenAmountOut.Amount.Sign() > 0 {
				t.Errorf("%s: reference failed but simulator accepted out=%s",
					label, simRes.TokenAmountOut.Amount)
			}
			continue
		}
		if simErr != nil || simRes == nil {
			t.Logf("%s: reference out=%s but simulator skipped (%v)",
				label, quoterOut, simErr)
			continue
		}

		bps := bpsDiff(quoterOut, simRes.TokenAmountOut.Amount)
		t.Logf("%s quote=%s sim=%s bps=%d", label, quoterOut, simRes.TokenAmountOut.Amount, bps)
		require.LessOrEqualf(t, bps, int64(200), "%s: sim/quote diff too high", label)
	}
}

func bpsDiff(quoter, sim *big.Int) int64 {
	if quoter.Sign() == 0 {
		return 0
	}
	diff := new(big.Int).Abs(new(big.Int).Sub(quoter, sim))
	return new(big.Int).Div(new(big.Int).Mul(diff, bignumber.BasisPoint), quoter).Int64()
}
