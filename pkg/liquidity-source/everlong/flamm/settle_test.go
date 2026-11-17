package everlongflamm

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"os"
	"slices"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

// settleVectorsSHA256 pins testdata/settle_vectors.json, which Tycho's tests read too: the same bytes in both repos.
const settleVectorsSHA256 = "9bf1e7bcceee7a162c0a0bba8ff05c0c60c929b52e51d1fb1b829b011f94473c"

// vectorCase is one settle case as the vectors write it, every number a decimal string but the decimals and the
// probes' statuses. A null request is one not read.
type vectorCase struct {
	Name     string   `json:"name"`
	Rule     string   `json:"rule"`
	Sale     bool     `json:"sale"`
	Tiers    []string `json:"tiers"`
	Trusted  bool     `json:"trusted"`
	CurveFee string   `json:"curveFee"`
	Request  *struct {
		BandWad string   `json:"bandWad"`
		UsdIn   string   `json:"usdIn"`
		UsdOut  string   `json:"usdOut"`
		Legs    []string `json:"legs"`
		BidWad  string   `json:"bidWad"`
	} `json:"request"`
	RowBand string  `json:"rowBand"`
	DecIn   uint8   `json:"decIn"`
	DecOut  uint8   `json:"decOut"`
	Probes  [][]any `json:"probes"` // ask, used, out, feeWad, status
	Want    struct {
		X      []string `json:"x"`
		Y      []string `json:"y"`
		Flat   []bool   `json:"flat"`
		Capped bool     `json:"capped"`
	} `json:"want"`
	Note string `json:"note"`
}

type rowOkCase struct {
	Name     string `json:"name"`
	RowBand  string `json:"rowBand"`
	Live     string `json:"live"`
	SpreadDn string `json:"spreadDn"`
	UpCap    string `json:"upCap"`
	DnCap    string `json:"dnCap"`
	Want     bool   `json:"want"`
}

func dec(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok, s)
	return v
}

func decs(t *testing.T, v []string) []*big.Int {
	t.Helper()
	out := make([]*big.Int, len(v))
	for k, s := range v {
		out[k] = dec(t, s)
	}
	return out
}

func (c vectorCase) input(t *testing.T) settleIn {
	t.Helper()
	in := settleIn{rule: map[string]settleRule{"EVERLONG": ruleEverlong, "STEP": ruleStep}[c.Rule],
		tiers: decs(t, c.Tiers), rowBand: dec(t, c.RowBand), sale: c.Sale, decIn: c.DecIn, decOut: c.DecOut,
		trusted: c.Trusted, curveFee: dec(t, c.CurveFee)}
	require.Contains(t, []string{"EVERLONG", "STEP"}, c.Rule)
	if r := c.Request; r != nil {
		require.Len(t, r.Legs, 8)
		in.words = pairWords{band: dec(t, r.BandWad), usdIn: dec(t, r.UsdIn), usdOut: dec(t, r.UsdOut),
			bid: dec(t, r.BidWad)}
		copy(in.words.legs[:], decs(t, r.Legs))
	}
	for _, p := range c.Probes {
		require.Len(t, p, 5)
		st := probeStatus(p[4].(float64))
		in.probes = append(in.probes, sampled{ask: dec(t, p[0].(string)), used: dec(t, p[1].(string)),
			out: dec(t, p[2].(string)), fee: dec(t, p[3].(string)), status: st})
	}
	return in
}

// check settles the case and holds it to its want, and to what settle may do: only drop a probe or lower an output.
func (c vectorCase) check(t *testing.T) {
	t.Helper()
	in := c.input(t)
	got := settle(in)
	require.Equal(t, c.Want.X, bigStrings(got.x), "%s: x", c.Name)
	require.Equal(t, c.Want.Y, bigStrings(got.y), "%s: y", c.Name)
	require.Equal(t, flags(c.Want.Flat), flags(got.flat), "%s: flat", c.Name)
	require.Equal(t, c.Want.Capped, got.capped, "%s: capped", c.Name)
	for k := range got.x {
		i := slices.IndexFunc(in.probes, func(p sampled) bool { return live(p) && p.used.Cmp(got.x[k]) == 0 })
		require.GreaterOrEqual(t, i, 0, "%s: point %d is a probe's", c.Name, k)
		require.LessOrEqual(t, got.y[k].Cmp(in.probes[i].out), 0, "%s: point %d over its probe", c.Name, k)
		if k > 0 {
			require.Positive(t, got.x[k].Cmp(got.x[k-1]), c.Name)
			require.GreaterOrEqual(t, got.y[k].Cmp(got.y[k-1]), 0, c.Name)
		}
	}
	require.Equal(t, c.Probes, c.reread(t, in.probes), "%s: settle leaves its input as it was", c.Name)
}

// reread writes the probes back in the vectors' form.
func (c vectorCase) reread(t *testing.T, probes []sampled) [][]any {
	t.Helper()
	out := make([][]any, len(probes))
	for k, p := range probes {
		out[k] = []any{p.ask.String(), p.used.String(), p.out.String(), p.fee.String(), float64(p.status)}
	}
	return out
}

func bigStrings(v []*big.Int) []string {
	out := make([]string, len(v))
	for k, x := range v {
		out[k] = x.String()
	}
	return out
}

// flags is a nil-safe view of flags: an empty ladder's are none either way.
func flags(v []bool) []bool {
	if len(v) == 0 {
		return nil
	}
	return v
}

func TestSettleMatchesTheSharedVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/settle_vectors.json")
	require.NoError(t, err)
	sum := sha256.Sum256(raw)
	require.Equal(t, settleVectorsSHA256, hex.EncodeToString(sum[:]), "the file both clients read, byte for byte")
	var v struct {
		Cases []vectorCase `json:"cases"`
		RowOk []rowOkCase  `json:"rowOk"`
	}
	require.NoError(t, json.Unmarshal(raw, &v))
	require.Len(t, v.Cases, 76)
	require.Len(t, v.RowOk, 13)
	for _, c := range v.Cases {
		t.Run(c.Name, c.check)
	}
	for _, c := range v.RowOk {
		row := hookRow{band: dec(t, c.RowBand), spreadDn: dec(t, c.SpreadDn), upCap: dec(t, c.UpCap),
			dnCap: dec(t, c.DnCap)}
		require.Equal(t, c.Want, rowOk(row, dec(t, c.Live)), c.Name)
	}
}

// The rule's named cases, in the vectors' form: each a clause of the rule on a sale of cbBTC for USDC at $100,000,
// band 8 %, the row's 5 %, its tiers 1.75 % and 1.3 % and a curve label of 1.5 %, a tier fill paying 995 USDC units
// a sat and a curve fill 990.
func TestSettleNamedCases(t *testing.T) {
	raw, err := os.ReadFile("testdata/settle_named.json")
	require.NoError(t, err)
	var v struct {
		Cases []vectorCase `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &v))
	require.Len(t, v.Cases, 21)
	for _, c := range v.Cases {
		t.Run(c.Name, c.check)
	}
}

func TestRowOkReadsTheRowAtTheTighterBand(t *testing.T) {
	r := hookRow{spread: big.NewInt(17_500_000_000_000_000), spreadDn: big.NewInt(13_000_000_000_000_000),
		upCap: big.NewInt(5_000_000_000_000_000), dnCap: big.NewInt(20_000_000_000_000_000),
		band: big.NewInt(50_000_000_000_000_000)}
	require.True(t, rowOk(r, big.NewInt(80_000_000_000_000_000)))
	require.False(t, rowOk(r, big.NewInt(20_000_000_000_000_000)), "the live band leaves the lever-down's cap no room")
	require.False(t, rowOk(r, nil), "no live band")
	require.False(t, rowOk(hookRow{}, big.NewInt(80_000_000_000_000_000)), "no row")
	require.Equal(t, []*big.Int{r.spread, r.spreadDn}, r.tiers())
	require.Empty(t, hookRow{spread: new(big.Int), spreadDn: big.NewInt(1)}.tiers(), "a zero spread: no leverage")
}
