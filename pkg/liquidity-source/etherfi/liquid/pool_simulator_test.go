package liquid

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	lhype  = "0x5748ae796ae46a4f1348a1693de4b50560485562"
	whype  = "0x5555555555555555555555555555555555555555"
	sthype = "0xffaa4a3d97fe9107cef8a3f48c069f577ff76cc1"
	// accountant.getRateInQuoteSafe(WHYPE) on hyperevm at block 49190448.
	lhypeRate = "1030663339500000000"
)

func newLHypeSim(t *testing.T, mutate func(*Extra)) *PoolSimulator {
	rate, _ := new(big.Int).SetString(lhypeRate, 10)
	extra := Extra{AssetData: []Asset{{AllowDeposits: true}}, RateInQuote: []*big.Int{rate}}
	if mutate != nil {
		mutate(&extra)
	}
	extraBytes, _ := json.Marshal(extra)
	staticBytes, _ := json.Marshal(StaticExtra{
		Teller:       hexAddr("0xfd83c1ca0c04e096d129275126fade1dc45bf4f0"),
		SimpleTeller: true,
	})
	sim, err := NewPoolSimulator(entity.Pool{
		Address:     lhype,
		Exchange:    "etherfi-liquid",
		Type:        DexType,
		Reserves:    []string{unlimitedReserve, unlimitedReserve},
		Tokens:      []*entity.PoolToken{{Address: lhype, Decimals: 18}, {Address: whype, Decimals: 18}},
		Extra:       string(extraBytes),
		StaticExtra: string(staticBytes),
	})
	require.NoError(t, err)
	return sim
}

func calc(sim *PoolSimulator, amt string) (*pool.CalcAmountOutResult, error) {
	in, _ := new(big.Int).SetString(amt, 10)
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: whype, Amount: in},
		TokenOut:      lhype,
	})
}

// Expected shares are the on-chain teller.deposit results (anvil fork of hyperevm block 49190448).
func TestCalcAmountOut_MatchesOnChainTeller(t *testing.T) {
	sim := newLHypeSim(t, nil)
	for in, want := range map[string]string{
		"1000000000000000000": "970248927729518800",
		"1000000000000000001": "970248927729518801", // floors, not rounds
		"7":                   "6",
	} {
		res, err := calc(sim, in)
		require.NoError(t, err)
		require.Equal(t, want, res.TokenAmountOut.Amount.String(), "in=%s", in)
		require.Equal(t, tellerDirectGas, res.Gas)
	}
}

func TestCalcAmountOut_RejectsWhatTellerReverts(t *testing.T) {
	for name, mutate := range map[string]func(*Extra){
		"teller paused":    func(e *Extra) { e.IsTellerPaused = true },
		"share lock":       func(e *Extra) { e.ShareLockPeriod = 1 },
		"asset removed":    func(e *Extra) { e.AssetData[0].AllowDeposits = false },
		"accountant pause": func(e *Extra) { e.RateInQuote[0] = nil },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := calc(newLHypeSim(t, mutate), "1000000000000000000")
			require.Error(t, err)
		})
	}
}

func TestSimpleTellerRouting(t *testing.T) {
	sim := newLHypeSim(t, nil)
	require.Equal(t, []string{whype}, sim.CanSwapTo(lhype))
	require.Equal(t, []string{lhype}, sim.CanSwapFrom(whype))
	require.Nil(t, sim.CanSwapFrom(lhype))
	// The vault pulls WHYPE in enter(), so the executor must approve the vault, not the teller.
	meta := sim.GetMetaInfo(whype, lhype).(MetaInfo)
	require.Equal(t, hexAddr(lhype), meta.ApprovalAddress)
	require.Equal(t, hexAddr("0xfd83c1ca0c04e096d129275126fade1dc45bf4f0"), meta.Teller)
	require.Equal(t, hexAddr("0x0"), meta.LiquidRefer)
}

func hexAddr(s string) common.Address { return common.HexToAddress(s) }
