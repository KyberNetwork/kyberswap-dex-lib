package prmfun

import (
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

const (
	referenceCurve     = "0xaea1eaf948e97581fbe9d8dea2951c327669af57" // ETH-paired, Phase.Trading
	referenceMemeToken = "0x54c1fa485f182a17b3ba9190e7ed481b6b60109e"
	graduatedCurve     = "0x0ea38ACC640A159F2cEbE5439A54D977ded64158" // $PRM's MemeCurve, Phase.Graduated
	graduatedMemeToken = "0xf24f8F6b08fE87CF062E833a732AD7F636064BC8"
)

func testPool(curveAddr, memeToken string) entity.Pool {
	staticExtraBytes, _ := json.Marshal(StaticExtra{
		RouterAddress:  "0x08A59435c8359A45F4F5dC8D91DF893Cc33DaF29",
		CurveAddress:   curveAddr,
		MemeToken:      memeToken,
		GraduationDesk: "4200000000000000000",
	})
	return entity.Pool{
		Address:     curveAddr,
		Exchange:    DexType,
		Type:        DexType,
		Tokens:      []*entity.PoolToken{{Address: testDeskToken}, {Address: memeToken}},
		Reserves:    []string{"0", "0"},
		StaticExtra: string(staticExtraBytes),
	}
}

func TestGetNewPoolState_TradingPool(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").SetMulticallContract(multicall3)
	tracker, err := NewPoolTracker(&Config{ChainId: 4663}, rpcClient)
	require.NoError(t, err)

	p, err := tracker.GetNewPoolState(t.Context(), testPool(referenceCurve, referenceMemeToken), pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	t.Logf("phase=%d virtualMeme=%s virtualDesk=%s memeSold=%s deskRaised=%s",
		extra.Phase, extra.VirtualMeme, extra.VirtualDesk, extra.MemeSold, extra.DeskRaised)

	require.Equal(t, PhaseTrading, extra.Phase)
	// derived reserves must equal the observed on-chain values (virtualMeme,
	// virtualDesk respectively - explorer.md's math_checks used curve-state order
	// [virtualMeme, virtualDesk], not the pool's own token index order).
	require.Equal(t, "1062330437710576052235807612", extra.VirtualMeme.Dec())
	require.Equal(t, "1405714531301211559", extra.VirtualDesk.Dec())

	// p.Reserves follows this pool type's own token index convention (0=desk, 1=meme).
	require.Equal(t, "1405714531301211559", p.Reserves[0])
	require.Equal(t, "1062330437710576052235807612", p.Reserves[1])
}

// TestGetNewPoolState_GraduatedPool exercises the "left Trading phase" branch against
// $PRM's real MemeCurve, which is genuinely graduated.
func TestGetNewPoolState_GraduatedPool(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").SetMulticallContract(multicall3)
	tracker, err := NewPoolTracker(&Config{ChainId: 4663}, rpcClient)
	require.NoError(t, err)

	p, err := tracker.GetNewPoolState(t.Context(), testPool(graduatedCurve, graduatedMemeToken), pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.Equal(t, PhaseGraduated, extra.Phase)
	require.Equal(t, []string{"0", "0"}, []string(p.Reserves))
}
