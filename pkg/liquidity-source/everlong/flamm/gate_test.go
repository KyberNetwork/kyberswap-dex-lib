package everlongflamm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// gateFxInt decodes a quoted signed decimal.
type gateFxInt struct{ gateInt }

func (x *gateFxInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	x.Neg = strings.HasPrefix(s, "-")
	if err := x.Abs.SetFromDecimal(strings.TrimPrefix(s, "-")); err != nil {
		return err
	}
	if x.Abs.IsZero() {
		x.Neg = false
	}
	return nil
}

type gateFxRes struct {
	V   *uint256.Int `json:"v"`
	Err string       `json:"err"`
	Ok  bool         `json:"ok"`
}

type gateFxCase struct {
	In struct {
		Scale          []uint256.Int `json:"scale"`
		Liquid         []uint256.Int `json:"liquid"`
		Supplied       []uint256.Int `json:"supplied"`
		Debt           []uint256.Int `json:"debt"`
		PriceWad       []uint256.Int `json:"priceWad"`
		CrossWad       []uint256.Int `json:"crossWad"`
		Physical       uint256.Int   `json:"physical"`
		Posted         uint256.Int   `json:"posted"`
		LtvWad         uint256.Int   `json:"ltvWad"`
		PhiWad         uint256.Int   `json:"phiWad"`
		RoomEpsilonWad uint256.Int   `json:"roomEpsilonWad"`
		QAny           []bool        `json:"qAny"`
		QDebt          []uint256.Int `json:"qDebt"`
		QColl          []uint256.Int `json:"qColl"`
	} `json:"in"`
	NetPW []struct {
		V   *gateFxInt `json:"v"`
		Err string     `json:"err"`
	} `json:"netPW"`
	ExposurePW        gateFxRes   `json:"exposurePW"`
	HeadOf            []gateFxRes `json:"headOf"`
	RoomNative        []gateFxRes `json:"roomNative"`
	RequiredPostedAll gateFxRes   `json:"requiredPostedAll"`
	NavAt             gateFxRes   `json:"navAt"`
	Context           struct {
		Liquid    uint256.Int `json:"liquid"`
		Supplied  uint256.Int `json:"supplied"`
		Debt      uint256.Int `json:"debt"`
		LoanCount uint256.Int `json:"loanCount"`
		Err       string      `json:"err"`
	} `json:"context"`
	StructuralDistWad uint256.Int `json:"structuralDistWad"`
	RoomWadNeg        uint256.Int `json:"roomWadNeg"`
	AssertGate        gateFxRes   `json:"assertGate"`
	Anchor            struct {
		U0          []gateFxInt `json:"u0"`
		Gross0      uint256.Int `json:"gross0"`
		Quarantined bool        `json:"quarantined"`
	} `json:"anchor"`
	Entry gateFxRes `json:"entry"`
	Exit  gateFxRes `json:"exit"`
	Post  struct {
		Physical  uint256.Int `json:"physical"`
		PostedAdj uint256.Int `json:"postedAdj"`
		DebtAdj   []gateFxInt `json:"debtAdj"`
	} `json:"post"`
}

// gateFakeRouter answers positions / quarantine from fixed per-asset figures, as the harness's mock router does.
type gateFakeRouter struct {
	sup, debt []uint256.Int
	posted    uint256.Int
	q         []mmQuarantine
}

func (f *gateFakeRouter) positions(uint64) (mmPositions, error) {
	return mmPositions{Coll: make([]uint256.Int, len(f.sup)), Sup: f.sup, Debt: f.debt, TotalColl: f.posted}, nil
}

func (f *gateFakeRouter) quarantine(idx uint8, _ uint64) (mmQuarantine, error) {
	return f.q[idx], nil
}

func gateCheckRes(t *testing.T, want gateFxRes, got *uint256.Int, err error, msg string) {
	t.Helper()
	if want.Err != "" {
		require.ErrorIs(t, err, mmRevertError(t, want.Err), msg)
		return
	}
	require.NoError(t, err, msg)
	if got != nil {
		require.Equal(t, want.V.Dec(), got.Dec(), msg)
	}
}

func gateIntString(x gateInt) string {
	if x.Neg {
		return "-" + x.Abs.Dec()
	}
	return x.Abs.Dec()
}

// TestGateMath replays FLAMMGateLib (commit 80abd43) over 400 seeded books: netPW, exposure, headOf, roomNative,
// the posting law, NAV, the hook context, the structural floor, roomWad on a surplus, and the storage composites
// (assertGate, anchor, assertEntryGate including the quarantined frame, assertExitNotWorsened) against a
// post-flow book.
func TestGateMath(t *testing.T) {
	t.Parallel()
	var fx struct {
		Cases []gateFxCase `json:"cases"`
	}
	mmReadFixture(t, "gate_math.json.gz", &fx)
	require.Equal(t, 400, len(fx.Cases))
	for k, c := range fx.Cases {
		in := &c.In
		n := len(in.Scale)
		b := gateBook{Physical: in.Physical, Posted: in.Posted, Legs: make([]gateLeg, n)}
		pool := &gatePool{Physical: in.Physical, LtvWad: in.LtvWad, PhiWad: in.PhiWad, RoomEpsilonWad: in.RoomEpsilonWad}
		fr := &gateFakeRouter{posted: in.Posted, sup: in.Supplied, debt: in.Debt}
		for i := 0; i < n; i++ {
			b.Legs[i] = gateLeg{Liquid: in.Liquid[i], Supplied: in.Supplied[i], Debt: in.Debt[i], Scale: in.Scale[i],
				PriceWad: in.PriceWad[i], CrossWad: in.CrossWad[i]}
			pool.Loans = append(pool.Loans, gateLoanCfg{Scale: in.Scale[i], Liquid: in.Liquid[i]})
			pool.PriceWad = append(pool.PriceWad, in.PriceWad[i])
			pool.CrossWad = append(pool.CrossWad, in.CrossWad[i])
			fr.q = append(fr.q, mmQuarantine{Any: in.QAny[i], FrozenDebt: in.QDebt[i], FrozenColl: in.QColl[i]})
		}
		name := fmt.Sprintf("case %d", k)
		for i := 0; i < n; i++ {
			got, err := gateNetPW(&b.Legs[i])
			if c.NetPW[i].Err != "" {
				require.ErrorIs(t, err, mmRevertError(t, c.NetPW[i].Err), name)
				continue
			}
			require.NoError(t, err, name)
			require.Equal(t, gateIntString(c.NetPW[i].V.gateInt), gateIntString(got), name+" netPW")
		}
		u, err := gateExposurePW(&b)
		gateCheckRes(t, c.ExposurePW, &u, err, name+" exposurePW")
		if err != nil {
			u.Clear()
		}
		for i := 0; i < n; i++ {
			h, err := gateHeadOf(&b, i, &u, &in.LtvWad)
			gateCheckRes(t, c.HeadOf[i], &h, err, fmt.Sprintf("%s headOf[%d]", name, i))
			rn, err := gateRoomNative(pool, &b, i, &u)
			gateCheckRes(t, c.RoomNative[i], &rn, err, fmt.Sprintf("%s roomNative[%d]", name, i))
		}
		need, err := gateRequiredPostedAll(&b, &in.LtvWad)
		gateCheckRes(t, c.RequiredPostedAll, &need, err, name+" requiredPostedAll")
		nav, err := gateNavAt(&b)
		gateCheckRes(t, c.NavAt, &nav, err, name+" navAt")
		ctx, err := gateContext(&b, &in.PriceWad[0], 1234, uint256.NewInt(5e18))
		if c.Context.Err != "" {
			require.ErrorIs(t, err, mmRevertError(t, c.Context.Err), name)
		} else {
			require.NoError(t, err, name)
			mmEq(t, &c.Context.Liquid, &ctx.LiquidLoanAsset, name+" context.liquid")
			mmEq(t, &c.Context.Supplied, &ctx.SuppliedLoanAsset, name+" context.supplied")
			mmEq(t, &c.Context.Debt, &ctx.DebtLoanAsset, name+" context.debt")
			require.Equal(t, c.Context.LoanCount.Uint64(), uint64(ctx.LoanCount))
		}
		var lltv uint256.Int
		lltv.Add(&in.LtvWad, &in.PhiWad).Rsh(&lltv, 1).AddUint64(&lltv, 1e17)
		sd := gateStructuralDistWad(&in.LtvWad, &lltv)
		mmEq(t, &c.StructuralDistWad, &sd, name+" structuralDistWad")
		var neg gateInt
		neg.Neg = true
		neg.Abs.Mul(&in.Physical, uint256.NewInt(1e10))
		if neg.Abs.IsZero() {
			neg.Neg = false
		}
		rw, err := gateRoomWad(neg, &in.Physical, uint256.NewInt(7e14), &in.LtvWad, &in.PhiWad)
		require.NoError(t, err)
		mmEq(t, &c.RoomWadNeg, &rw, name+" roomWadNeg")

		// storage composites against the mock router
		err = gateAssertGate(pool, fr, 0)
		gateCheckRes(t, c.AssertGate, nil, err, name+" assertGate")
		u0, g0, q0, err := gateAnchor(pool, fr, 0)
		require.NoError(t, err, name)
		require.Equal(t, c.Anchor.Quarantined, q0, name)
		mmEq(t, &c.Anchor.Gross0, &g0, name+" gross0")
		for i := range u0 {
			require.Equal(t, gateIntString(c.Anchor.U0[i].gateInt), gateIntString(u0[i]), name+" u0")
		}
		post := pool.clone()
		post.Physical = c.Post.Physical
		pr := &gateFakeRouter{sup: in.Supplied, q: fr.q, debt: make([]uint256.Int, n)}
		pr.posted.Add(&in.Posted, &c.Post.PostedAdj)
		for i := 0; i < n; i++ {
			adj := c.Post.DebtAdj[i].gateInt
			d, err := gateInt{Abs: in.Debt[i]}.add(adj)
			require.NoError(t, err, name)
			if !d.Neg {
				pr.debt[i] = d.Abs
			}
		}
		err = gateAssertEntryGate(post, pr, 0, u0, g0, q0)
		gateCheckRes(t, c.Entry, nil, err, name+" assertEntryGate")
		err = gateAssertExitNotWorsened(post, pr, 0, u0, g0)
		gateCheckRes(t, c.Exit, nil, err, name+" assertExitNotWorsened")
	}
}

// TestGateIntEdges replays FLAMMGateLib (80abd43, via-IR) over the int256 edges: casts wrapping at 2^255, checked
// add / sub / negation at type(int256).min in netL18, netPW, roomWad, headOf, _readableU and the quarantined entry
// frame, Math.mulDiv's rounded-up `+= 1` overflow, and the checked epsilon shave (testdata/gen/GateIntEdges.t.sol).
func TestGateIntEdges(t *testing.T) {
	t.Parallel()
	var rows []finGateRow
	loadGzFixture(t, "gate_int_edges.json.gz", &rows)
	require.Equal(t, 20, len(rows)-1)
	finGateReplay(t, "gate-edges", rows)
}
