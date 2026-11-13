package oneinchaqua

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// The case matrix is a list of strategies (a program, a router variant and the raw balances it is shipped
// with) and a list of cells (a strategy, a direction and an amount). Both lists are generated in a fixed
// order: the router oracle file holds one result per cell in that order.

var (
	matrixToken0 = common.HexToAddress("0x111111111117dC0aa78b770fA6A738034120C302") // 1INCH, the lower address
	matrixToken1 = common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F") // DAI
)

type matrixStrategy struct {
	Name    string
	Group   string
	Program []byte
	Launch  bool
	Raw     [2]*uint256.Int
	// WantParse is the discard reason when the program must be refused; empty when it must run.
	WantParse DiscardReason
	// Unshippable marks raw balances above uint248: the registry cannot hold them, so no router answer
	// exists and the interpreter is compared with the math/big model only.
	Unshippable bool
}

type matrixCell struct {
	Id       string
	Strategy int
	In       int
	Amount   *uint256.Int
	// Site names the checked-arithmetic site an overflow vector reaches ("" otherwise).
	Site string
}

type matrix struct {
	Strategies []matrixStrategy
	Cells      []matrixCell
}

// matrixMaker ships every strategy of the matrix: one account, so the fork fetches one account from its
// upstream. The orders differ by a nonce in maker traits bits 224 and up, which the router never reads.
var matrixMaker = common.HexToAddress("0x00000000000000000000000000000000A11CE000")

func (m *matrix) maker(int) common.Address { return matrixMaker }

func (m *matrix) order(i int) *Order {
	traits := aquaTraits()
	traits[3] |= uint64(i+1) << 32
	return &Order{Maker: matrixMaker, Traits: traits, Data: m.Strategies[i].Program}
}

func (m *matrix) app(i int) common.Address {
	if m.Strategies[i].Launch {
		return testLaunch
	}
	return testRouter
}

func (m *matrix) addStrategy(s matrixStrategy) int {
	for i := range 2 {
		if s.Raw[i] == nil {
			s.Raw[i] = new(uint256.Int)
		}
	}
	m.Strategies = append(m.Strategies, s)
	return len(m.Strategies) - 1
}

// addCells adds both directions of a strategy for the amounts.
func (m *matrix) addCells(s int, dirs []int, amounts []*uint256.Int, site string) {
	for _, d := range dirs {
		for _, a := range amounts {
			id := fmt.Sprintf("%s/%s/dir%d/amt%s", m.Strategies[s].Group, m.Strategies[s].Name, d, a.Dec())
			m.Cells = append(m.Cells, matrixCell{Id: id, Strategy: s, In: d, Amount: a, Site: site})
		}
	}
}

var bothDirs = []int{0, 1}

func amts(v ...string) []*uint256.Int {
	out := make([]*uint256.Int, len(v))
	for i, s := range v {
		out[i] = u(s)
	}
	return out
}

type wrapperKind byte

const (
	wrapC wrapperKind = 'C'
	wrapF wrapperKind = 'F'
	wrapP wrapperKind = 'P'
)

// wrapperInstruction returns the nth wrapper of its kind with parameters that differ per position, so a
// swapped order changes the result.
func wrapperInstruction(k wrapperKind, nth int) []byte {
	switch k {
	case wrapC:
		ranges := [][2]uint64{{1, 4}, {1, 2}, {2, 8}} // sqrt prices of ratios 1..4, 1..2 and 2..8
		r := ranges[nth]
		return iConcentrate(sqrtPrice(e18(r[0])), sqrtPrice(e18(r[1])))
	case wrapF:
		return iFlat([]uint32{10_000_000, 20_000_000, 3_000_000}[nth])
	default:
		return iProtocol([]uint32{5_000_000, 30_000_000, 1_000_000}[nth], testRecipient)
	}
}

// buildProgram turns a wrapper sequence like "PCF" into a program: salt variant, wrappers, xycSwap, salt.
func buildProgram(wrappers string, salt string) []byte {
	var parts [][]byte
	if salt == "lead" || salt == "both" {
		parts = append(parts, iSalt(0xAA, 0x21, 0xBB))
	}
	count := map[wrapperKind]int{}
	for _, c := range wrappers {
		k := wrapperKind(c)
		parts = append(parts, wrapperInstruction(k, count[k]))
		count[k]++
	}
	parts = append(parts, iSwap())
	if salt == "trail" || salt == "both" {
		parts = append(parts, iSalt(0xCC))
	}
	return prog(parts...)
}

func wrapperSequences(maxLen int) []string {
	seqs := []string{""}
	frontier := []string{""}
	for range maxLen {
		var next []string
		for _, s := range frontier {
			for _, k := range "CFP" {
				next = append(next, s+string(k))
			}
		}
		seqs = append(seqs, next...)
		frontier = next
	}
	return seqs
}

type rawProfile struct {
	Name string
	Raw  [2]*uint256.Int
}

var (
	profileA = rawProfile{"A", [2]*uint256.Int{e18(1000), e18(2000)}}
	profileB = rawProfile{"B", [2]*uint256.Int{new(uint256.Int), e18(5000)}}
)

func buildMatrix() *matrix {
	m := &matrix{}
	m.addShapes()
	m.addRouterVariants()
	m.addIllegal()
	m.addBoundaries()
	m.addRawCap()
	m.addOverflow()
	return m
}

// addShapes: every legal ordering of up to three wrappers, with and without salt, both directions, two
// balance profiles and three amounts.
func (m *matrix) addShapes() {
	for _, w := range wrapperSequences(3) {
		for _, salt := range []string{"none", "lead", "trail"} {
			for _, p := range []rawProfile{profileA, profileB} {
				name := fmt.Sprintf("%s-%s-%s", orDash(w), salt, p.Name)
				s := m.addStrategy(matrixStrategy{Name: name, Group: "shape", Program: buildProgram(w, salt), Raw: p.Raw})
				m.addCells(s, bothDirs, amts("1", "1000000000000000000", "123456789012345678901"), "")
			}
		}
	}
}

func orDash(s string) string {
	if s == "" {
		return "xyc"
	}
	return s
}

var liveShapes = []struct{ name, wrappers, salt string }{
	{"concentrate-flat-xyc", "CF", "none"},
	{"protocol-concentrate-flat-xyc-salt", "PCF", "trail"},
	{"flat-concentrate-xyc", "FC", "none"},
	{"salt-protocol2-flat-xyc", "PPF", "lead"},
	{"xyc", "", "none"},
	{"salt-xyc", "", "lead"},
}

// addRouterVariants: the live shapes on the current and the launch router.
func (m *matrix) addRouterVariants() {
	for _, launch := range []bool{false, true} {
		for _, ls := range liveShapes {
			for _, p := range []rawProfile{profileA, profileB} {
				name := fmt.Sprintf("%s-%s-%s", ls.name, map[bool]string{false: "current", true: "launch"}[launch], p.Name)
				s := m.addStrategy(matrixStrategy{Name: name, Group: "router", Program: buildProgram(ls.wrappers, ls.salt), Launch: launch, Raw: p.Raw})
				m.addCells(s, bothDirs, amts("1", "1000000000000000000", "123456789012345678901"), "")
			}
		}
	}
}

func truncated(program []byte, cut int) []byte { return program[:cut] }

// addIllegal: orderings and encodings the whitelist refuses. Every one is shipped, so the oracle records
// what the router does with it; the integration refuses all of them whatever the router answers.
func (m *matrix) addIllegal() {
	pmin, pmax := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	conc, flat, protocol := iConcentrate(pmin, pmax), iFlat(10_000_000), iProtocol(10_000_000, testRecipient)
	longProgram := prog(protocol, conc, flat, iSwap(), iSalt(1, 2, 3))
	add := func(name string, program []byte, reason DiscardReason) {
		s := m.addStrategy(matrixStrategy{Name: name, Group: "illegal", Program: program, Raw: profileA.Raw, WantParse: reason})
		m.addCells(s, []int{0}, amts("1000000000000000000"), "")
	}
	add("flat-after-xyc", prog(iSwap(), flat), ReasonProgramShape)
	add("protocol-after-xyc", prog(iSwap(), protocol), ReasonProgramShape)
	add("concentrate-after-xyc", prog(iSwap(), conc), ReasonProgramShape)
	add("fee-then-xyc-then-concentrate", prog(flat, iSwap(), conc), ReasonProgramShape)
	add("two-xyc", prog(iSwap(), iSwap()), ReasonProgramShape)
	add("two-xyc-with-wrappers", prog(conc, flat, iSwap(), iSwap()), ReasonProgramShape)
	add("no-xyc-empty-salt", prog(iSalt(1)), ReasonProgramShape)
	add("no-xyc-wrappers", prog(conc, flat), ReasonProgramShape)
	add("concentrate-only", prog(conc), ReasonProgramShape)
	add("empty", nil, ReasonProgramEmpty)
	add("xyc-args-1", prog(ins(opXYCSwap, 1)), ReasonBadArgs)
	for _, d := range []int{-1, 1} {
		add(fmt.Sprintf("flat-args%+d", d), prog(ins(opFlatFeeIn, make([]byte, 4+d)...), iSwap()), ReasonBadArgs)
		add(fmt.Sprintf("protocol-args%+d", d), prog(ins(opAquaProtocolFee, make([]byte, 24+d)...), iSwap()), ReasonBadArgs)
		add(fmt.Sprintf("concentrate-args%+d", d), prog(ins(opXYCConcentrate, make([]byte, 64+d)...), iSwap()), ReasonBadArgs)
	}
	for cut := 1; cut < len(longProgram); cut += 5 {
		if _, err := parseProgram(truncated(longProgram, cut), false); ReasonOf(err) != ReasonProgramTruncated {
			continue // a cut on an instruction boundary is a shorter program, refused for its shape
		}
		add(fmt.Sprintf("truncated-at-%d", cut), truncated(longProgram, cut), ReasonProgramTruncated)
	}
	add("flat-bps-1e9+1", prog(iFlat(feeBpsDenominator+1), iSwap()), ReasonFeeBps)
	add("protocol-bps-1e9+1", prog(iProtocol(feeBpsDenominator+1, testRecipient), iSwap()), ReasonFeeBps)
	add("sqrtPmin-eq-sqrtPmax", prog(iConcentrate(pmin, pmin), iSwap()), ReasonPriceBounds)
	add("sqrtPmin-gt-sqrtPmax", prog(iConcentrate(pmax, pmin), iSwap()), ReasonPriceBounds)
	add("sqrtPmin-zero-sqrtPmax-zero", prog(iConcentrate(new(uint256.Int), new(uint256.Int)), iSwap()), ReasonPriceBounds)
	add("guard-first", prog(guard(), iSwap()), ReasonTxOriginGuard)
	add("guard-middle", prog(flat, guard(), iSwap()), ReasonTxOriginGuard)
	add("guard-last", prog(iSwap(), guard()), ReasonTxOriginGuard)
	add("opcode-0x22", prog(ins(0x22), iSwap()), ReasonUnsupportedOpcode)
	add("opcode-0x17-unused-noop", prog(ins(0x17), iSwap()), ReasonUnsupportedOpcode)
	add("jump", prog(ins(0x0a, 0, 0), iSwap()), ReasonUnsupportedOpcode)
	add("zero-recipient-bps1-current", prog(iProtocol(1, common.Address{}), iSwap()), ReasonFeeRecipient)
	s := m.addStrategy(matrixStrategy{Name: "zero-recipient-bps1-launch", Group: "illegal", Program: prog(iProtocol(1, common.Address{}), iSwap()), Launch: true, Raw: profileA.Raw, WantParse: ReasonLaunchFeeRecipent})
	m.addCells(s, []int{0}, amts("1000000000000000000"), "")
	s = m.addStrategy(matrixStrategy{Name: "zero-recipient-bps0-launch", Group: "illegal", Program: prog(iProtocol(0, common.Address{}), iSwap()), Launch: true, Raw: profileA.Raw, WantParse: ReasonLaunchFeeRecipent})
	m.addCells(s, []int{0}, amts("1000000000000000000"), "")
	// bps 0 with a zero recipient is legal on the current router.
	s = m.addStrategy(matrixStrategy{Name: "zero-recipient-bps0-current", Group: "router", Program: prog(iProtocol(0, common.Address{}), iSwap()), Raw: profileA.Raw})
	m.addCells(s, bothDirs, amts("1000000000000000000"), "")
}

func (m *matrix) addBoundaries() {
	big18 := "1000000000000000000"
	// amount boundaries on the main live shape.
	s := m.addStrategy(matrixStrategy{Name: "cfx-A", Group: "amount", Program: buildProgram("CF", "none"), Raw: profileA.Raw})
	m.addCells(s, bothDirs, amts("1", "2", "3", "10", "1000", "999999999", "1000000000", "1000000001", big18, "999999999999999999999", "5000000000000000000000", "1000000000000000000000000"), "")
	// outputs that round to zero.
	for _, p := range []struct {
		name    string
		program []byte
	}{{"xyc", prog(iSwap())}, {"flat-xyc", prog(iFlat(10_000_000), iSwap())}, {"protocol-xyc", prog(iProtocol(10_000_000, testRecipient), iSwap())}} {
		s = m.addStrategy(matrixStrategy{Name: "zero-out-" + p.name, Group: "zeroout", Program: p.program, Raw: [2]*uint256.Int{u("1000000000000000000000"), uint256.NewInt(1)}})
		m.addCells(s, []int{0}, amts("1", "1000", "1000000000000000000000", "100000000000000000000000"), "")
	}
	// balances 0 and 1 on each side, before and after concentrate.
	var side = func(v string) *uint256.Int { return u(v) }
	vals := []string{"0", "1", "1000000000000000000"}
	for _, prog := range []struct {
		name string
		p    []byte
	}{{"xyc", prog(iSwap())}, {"cfx", buildProgram("CF", "none")}} {
		for _, a := range vals {
			for _, b := range vals {
				s = m.addStrategy(matrixStrategy{Name: fmt.Sprintf("%s-%s-%s", prog.name, a, b), Group: "balances", Program: prog.p, Raw: [2]*uint256.Int{side(a), side(b)}})
				m.addCells(s, bothDirs, amts("1", big18), "")
			}
		}
	}
	// fee bps boundaries, flat and protocol.
	for _, bps := range []uint32{0, 1, feeBpsDenominator - 1, feeBpsDenominator} {
		for _, k := range []struct {
			kind string
			mk   func(uint32) []byte
		}{{"flat", iFlat}, {"protocol", func(b uint32) []byte { return iProtocol(b, testRecipient) }}} {
			kind, mk := k.kind, k.mk
			s = m.addStrategy(matrixStrategy{Name: fmt.Sprintf("%s-%d", kind, bps), Group: "bps", Program: prog(mk(bps), iSwap()), Raw: profileA.Raw})
			m.addCells(s, bothDirs, amts("1", "7", "999", big18), "")
		}
	}
	// price-bound boundaries.
	type bound struct {
		name       string
		pmin, pmax *uint256.Int
	}
	for _, b := range []bound{
		{"0-2e18", new(uint256.Int), e18(2)},
		{"1-2", uint256.NewInt(1), uint256.NewInt(2)},
		{"pmax-1", u("1999999999999999999"), e18(2)},
		{"1e18-2e18", e18(1), e18(2)},
		{"1-2pow255", uint256.NewInt(1), new(uint256.Int).Lsh(uint256.NewInt(1), 255)},
		{"narrow", uint256.NewInt(1e18 - 1), uint256.NewInt(1e18 + 1)},
	} {
		for _, p := range []rawProfile{profileA, profileB, {"C", [2]*uint256.Int{e18(5000), new(uint256.Int)}}, {"D", [2]*uint256.Int{uint256.NewInt(1), u("1000000000000000000000000")}},
			{"E", [2]*uint256.Int{u("1000000000000000000000000"), uint256.NewInt(1)}}} {
			s = m.addStrategy(matrixStrategy{Name: b.name + "-" + p.Name, Group: "sqrtp", Program: prog(iConcentrate(b.pmin, b.pmax), iSwap()), Raw: p.Raw})
			m.addCells(s, bothDirs, amts("1", big18), "")
		}
	}
	// rounding: where ceil and floor differ.
	for _, r := range []struct {
		name    string
		program []byte
		raw     [2]*uint256.Int
		amounts []*uint256.Int
	}{
		{"flat-ceil", prog(iFlat(10_000_000), iSwap()), profileA.Raw, amts("1", "99", "100", "101", "999", "1000", "1001", "100000000000000001")},
		{"flat-ceil-bps1", prog(iFlat(1), iSwap()), profileA.Raw, amts("1", "999999999", "1000000000", "1000000001", "1999999999")},
		{"protocol-floor", prog(iProtocol(10_000_000, testRecipient), iSwap()), profileA.Raw, amts("1", "99", "100", "101", "999", "1000", "1001", "100000000000000001")},
		{"concentrate-ceil", prog(iConcentrate(uint256.NewInt(1e18-1), uint256.NewInt(1e18+1)), iSwap()), [2]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1000)}, amts("1", "2", "10", "500", "999", "1000", "100000")},
		{"concentrate-ceil-large", prog(iConcentrate(sqrtPrice(e18(1)), sqrtPrice(e18(4))), iSwap()), [2]*uint256.Int{uint256.NewInt(3), uint256.NewInt(7)}, amts("1", "2", "3", "5", "100")},
	} {
		s = m.addStrategy(matrixStrategy{Name: r.name, Group: "rounding", Program: r.program, Raw: r.raw})
		m.addCells(s, bothDirs, r.amounts, "")
	}
	// uint248 balances: the product amountIn*balanceOut and the sum balanceIn+amountIn.
	s = m.addStrategy(matrixStrategy{Name: "max248-both", Group: "uint248", Program: prog(iSwap()), Raw: [2]*uint256.Int{max248, max248}})
	m.addCells(s, bothDirs, amts("1"), "")
	m.addCells(s, bothDirs, amts(big18), "xyc.num")
	rawOut := new(uint256.Int).Lsh(uint256.NewInt(1), 180)
	s = m.addStrategy(matrixStrategy{Name: "max248-in", Group: "uint248", Program: prog(iSwap()), Raw: [2]*uint256.Int{max248, rawOut}})
	m.addCells(s, []int{0}, amts("1", "1000000000000000000000"), "")
}

// overflowVectors name every checked-arithmetic site with a concrete vector. A vector with raw balances
// above uint248 cannot be shipped.
type overflowVector struct {
	Site        string
	Program     []byte
	Raw         [2]*uint256.Int
	In          int
	Amount      *uint256.Int
	Unshippable bool
}

func overflowVectors() []overflowVector {
	max256 := new(uint256.Int).Not(new(uint256.Int))
	half := new(uint256.Int).Rsh(max256, 1)
	concProgram := func(pmin, pmax string) []byte { return prog(iConcentrate(u(pmin), u(pmax)), iSwap()) }
	// Inputs found by random search against the math/big model; token0 is the lower address, so
	// direction 0 has tokenIn lower (inLt true).
	return []overflowVector{
		{"xyc.num", prog(iSwap()), [2]*uint256.Int{e18(1), half}, 0, e18(10), true},
		{"xyc.num", prog(iSwap()), [2]*uint256.Int{e18(1), e18(1000)}, 0, max256, false},
		{"xyc.den", prog(iSwap()), [2]*uint256.Int{e18(1), uint256.NewInt(1)}, 0, new(uint256.Int).SubUint64(max256, 1), false},
		{"flat.mul", prog(iFlat(10_000_000), iSwap()), [2]*uint256.Int{e18(1), e18(1000)}, 0, max256, false},
		{"protocol.mul", prog(iProtocol(10_000_000, testRecipient), iSwap()), [2]*uint256.Int{e18(1), e18(1000)}, 0, max256, false},
		{"conc.beta.lt", concProgram("936523951040907692092419788024370259484074785569191370284613", "36127572523700336823722850999516127410201335483653968073943696"),
			[2]*uint256.Int{u("341193806949478780381580174557165135468320949838143328456"), u("3402896932133346566420654")}, 0, uint256.NewInt(1000), false},
		{"conc.beta.gt", concProgram("65", "3732811062"),
			[2]*uint256.Int{u("78620693316208647207465249781395141488712774364391293812566072234947"), u("766696884319665615006408758359752000153596936332856330449985317121379461")}, 0, uint256.NewInt(1000), false},
		{"conc.fourAC.mul", concProgram("1077171647979167619", "2506349857314932135884908470796191964628720938674867"),
			[2]*uint256.Int{u("71815454866502265434044201315525558170634578558821457"), u("4009137316531302415249616018681512101285479219622741845712707785")}, 1, uint256.NewInt(1000), false},
		{"conc.disc.square", concProgram("5611979085471245", "4040206021534379245616"),
			[2]*uint256.Int{u("54407457979048829015"), u("11290587235660575360086340642415281009270916803010")}, 0, uint256.NewInt(1000), false},
		{"conc.disc.add", concProgram("893996379", "61921770218525"),
			[2]*uint256.Int{u("334700011584859443943875080966634254569901606953"), u("26946899464649753696710166204")}, 1, uint256.NewInt(1000), false},
		{"conc.addIn.mul", concProgram("11050380066215589452074776985480136214511181511", "811149783243250300561687657954823312293103892968"),
			[2]*uint256.Int{u("15752146"), u("276051729109481")}, 1, uint256.NewInt(1000), false},
		{"conc.addOut.mulDiv", concProgram("312136993923529941143815648584621358062617324861780278873", "602771509678339220800955821617845748035144789357435590310"),
			[2]*uint256.Int{uint256.NewInt(1), u("1263603041934025768696041675103051029958331543233667710596")}, 0, uint256.NewInt(1000), false},
		{"conc.fourAC.mulDiv", concProgram("2", "2148507203689315581166308623623170170209374548621884841054"),
			[2]*uint256.Int{u("55296728325394067258190129225891429549127816511207631979945318093737006224985"), u("826618969637965619")}, 1, uint256.NewInt(1000), true},
		{"conc.balOut.add", concProgram("3295843703600144403740989580639355134023965613768836674166812893092403938", "12590701312818183159142317895849644692987919319386702368963608116644413993"),
			[2]*uint256.Int{new(uint256.Int), u("101255403379745167828515961193116107430182740547579958081801749296311691764253")}, 0, uint256.NewInt(1000), true},
	}
}

func (m *matrix) addOverflow() {
	for i, v := range overflowVectors() {
		name := fmt.Sprintf("%02d-%s", i, strings.ReplaceAll(v.Site, ".", "_"))
		s := m.addStrategy(matrixStrategy{Name: name, Group: "overflow", Program: v.Program, Raw: v.Raw, Unshippable: v.Unshippable})
		m.addCells(s, []int{v.In}, []*uint256.Int{v.Amount}, v.Site)
	}
}

// addRawCap: amounts around the point where the quote reaches the strategy's own out balance. Concentrate
// adds virtual out liquidity, so a quote can exceed what the strategy holds.
func (m *matrix) addRawCap() {
	raw := [2]*uint256.Int{e18(1), e18(1)}
	program := prog(iConcentrate(sqrtPrice(e18(1)), sqrtPrice(e18(4))), iFlat(10_000_000), iSwap())
	parsed, _ := parseProgram(program, false)
	lo, hi := uint256.NewInt(1), new(uint256.Int).Lsh(uint256.NewInt(1), 128)
	for lo.Lt(hi) {
		mid := new(uint256.Int).Add(lo, hi)
		mid.Rsh(mid, 1)
		r, err := runQuote(parsed, raw[0], raw[1], mid, true)
		if err == nil && !r.AmountOut.Lt(raw[1]) {
			hi = mid
		} else {
			lo.AddUint64(mid, 1)
		}
	}
	s := m.addStrategy(matrixStrategy{Name: "cfx-drain", Group: "rawcap", Program: program, Raw: raw})
	m.addCells(s, []int{0}, []*uint256.Int{new(uint256.Int).SubUint64(hi, 1), hi, new(uint256.Int).AddUint64(hi, 1), new(uint256.Int).Lsh(hi, 1)}, "")
}

// siteComputeL and siteConc are the math/big model with each checked-arithmetic site named: they return
// the site that overflows, or "" when the computation fits. Overflow vectors in the case matrix assert the
// site they claim to reach.
func siteComputeL(bLt, bGt, pmin, pmax *big.Int) (*big.Int, string) {
	ratio, ok := refMulDiv(pmin, refOne, pmax)
	if !ok {
		return nil, "conc.ratio"
	}
	alpha := new(big.Int).Sub(refOne, ratio)
	b1, ok1 := refMulDiv(bLt, pmin, refOne)
	if !ok1 {
		return nil, "conc.beta.lt"
	}
	b2, ok2 := refMulDiv(bGt, refOne, pmax)
	if !ok2 {
		return nil, "conc.beta.gt"
	}
	beta := new(big.Int).Add(b1, b2)
	if !fits(beta) {
		return nil, "conc.beta.sum"
	}
	four, ok := refMulDiv(new(big.Int).Lsh(alpha, 2), bLt, refOne)
	if !ok {
		return nil, "conc.fourAC.mulDiv"
	}
	four.Mul(four, bGt)
	if !fits(four) {
		return nil, "conc.fourAC.mul"
	}
	disc := new(big.Int).Mul(beta, beta)
	if !fits(disc) {
		return nil, "conc.disc.square"
	}
	disc.Add(disc, four)
	if !fits(disc) {
		return nil, "conc.disc.add"
	}
	num := new(big.Int).Add(beta, new(big.Int).Sqrt(disc))
	if !fits(num) {
		return nil, "conc.L.numSum"
	}
	l, ok := refMulDiv(num, refOne, new(big.Int).Lsh(alpha, 1))
	if !ok {
		return nil, "conc.L.mulDiv"
	}
	return l, ""
}

func siteConc(balIn, balOut, pmin, pmax *big.Int, inLt bool) string {
	bLt, bGt := balOut, balIn
	if inLt {
		bLt, bGt = balIn, balOut
	}
	l, site := siteComputeL(bLt, bGt, pmin, pmax)
	if site != "" {
		return site
	}
	var addIn, addOut *big.Int
	if inLt {
		prod := new(big.Int).Mul(l, refOne)
		if !fits(prod) {
			return "conc.addIn.mul"
		}
		addIn = refCeilDiv(prod, pmax)
		var ok bool
		addOut, ok = refMulDiv(l, pmin, refOne)
		if !ok {
			return "conc.addOut.mulDiv"
		}
	} else {
		prod := new(big.Int).Mul(l, pmin)
		if !fits(prod) {
			return "conc.addIn.mul"
		}
		addIn = refCeilDiv(prod, refOne)
		var ok bool
		addOut, ok = refMulDiv(l, refOne, pmax)
		if !ok {
			return "conc.addOut.mulDiv"
		}
	}
	if !fits(new(big.Int).Add(balIn, addIn)) {
		return "conc.balIn.add"
	}
	if !fits(new(big.Int).Add(balOut, addOut)) {
		return "conc.balOut.add"
	}
	return ""
}

// siteOfProgram names the first overflowing site of a program run on the model, or "".
func siteOfProgram(program []Instruction, balIn, balOut, amtIn *big.Int, inLt bool) string {
	balIn, balOut, amtIn = new(big.Int).Set(balIn), new(big.Int).Set(balOut), new(big.Int).Set(amtIn)
	for _, p := range program {
		switch p.Op {
		case opXYCConcentrate:
			pmin, pmax := p.SqrtPMin.ToBig(), p.SqrtPMax.ToBig()
			if site := siteConc(balIn, balOut, pmin, pmax, inLt); site != "" {
				return site
			}
			bLt, bGt := balOut, balIn
			if inLt {
				bLt, bGt = balIn, balOut
			}
			l, _ := refComputeL(bLt, bGt, pmin, pmax)
			if inLt {
				balIn.Add(balIn, refCeilDiv(new(big.Int).Mul(l, refOne), pmax))
				add, _ := refMulDiv(l, pmin, refOne)
				balOut.Add(balOut, add)
			} else {
				balIn.Add(balIn, refCeilDiv(new(big.Int).Mul(l, pmin), refOne))
				add, _ := refMulDiv(l, refOne, pmax)
				balOut.Add(balOut, add)
			}
		case opFlatFeeIn:
			prod := new(big.Int).Mul(amtIn, big.NewInt(int64(p.Bps)))
			if !fits(prod) {
				return "flat.mul"
			}
			amtIn.Sub(amtIn, refCeilDiv(prod, refBps))
		case opAquaProtocolFee:
			prod := new(big.Int).Mul(amtIn, big.NewInt(int64(p.Bps)))
			if !fits(prod) {
				return "protocol.mul"
			}
			amtIn.Sub(amtIn, prod.Quo(prod, refBps))
		case opXYCSwap:
			if !fits(new(big.Int).Mul(amtIn, balOut)) {
				return "xyc.num"
			}
			if !fits(new(big.Int).Add(balIn, amtIn)) {
				return "xyc.den"
			}
		}
	}
	return ""
}

func selectorOf(sig string) string { return hexutil.Encode(crypto.Keccak256([]byte(sig))[:4]) }

// routerErrors names the custom errors and panics the router answers a quote with, and the interpreter
// outcome each one must map to.
var routerErrors = map[string]string{
	"EVM error OutOfGas": "refused", // a jump back to the start of the program loops until the gas runs out
	"execution reverted": "refused", // a revert with no data: the router's calldata slice runs out of the program
	selectorOf("TakerTraitsAmountOutMustBeGreaterThanZero(uint256)"):                "zero_out",
	selectorOf("XYCSwapRequiresBothBalancesNonZero(uint256,uint256)"):               "zero_balance",
	selectorOf("FeeDynamicProtocolInvalidRecipient()"):                              "refused",
	selectorOf("FeeBpsOutOfRange(uint256)"):                                         "refused",
	selectorOf("RunLoopExcessiveCall(uint256,uint256)"):                             "refused",
	selectorOf("FeeShouldBeAppliedBeforeSwapAmountsComputation()"):                  "refused",
	selectorOf("XYCSwapRecomputeDetected()"):                                        "refused",
	selectorOf("ConcentrateShouldBeUsedBeforeSwapAmountsComputed(uint256,uint256)"): "refused",
	selectorOf("FeeBpsOutOfRange(uint32)"):                                          "refused",
	selectorOf("FeeMissingFeeBPS()"):                                                "refused",
	selectorOf("ProtocolFeeMissingFeeBPS()"):                                        "refused",
	selectorOf("ProtocolFeeMissingTo()"):                                            "refused",
	selectorOf("ConcentrateMissingSqrtPriceMin()"):                                  "refused",
	selectorOf("ConcentrateMissingSqrtPriceMax()"):                                  "refused",
}

func matrixTestName(c matrixCell) string { return strings.ReplaceAll(c.Id, "/", ".") }

// checkAgainstRouter compares the interpreter with one oracle result.
func checkAgainstRouter(t *testing.T, prog []Instruction, c matrixCell, raw [2]*uint256.Int, result string) {
	t.Helper()
	in, out := c.In, 1-c.In
	inLt := in == 0 // matrix token0 < token1
	got, err := runQuote(prog, raw[in], raw[out], c.Amount, inLt)
	kind, value, _ := strings.Cut(result, ":")
	switch {
	case result == "-": // not shippable: the math/big model stands in for the router
		wantIn, wantOut, _, ok := refRun(prog, raw[in].ToBig(), raw[out].ToBig(), c.Amount.ToBig(), inLt)
		if !ok {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		require.Equal(t, wantOut.String(), got.AmountOut.Dec())
		require.Equal(t, wantIn.String(), got.AmountIn.Dec())
	case kind == "o":
		require.NoError(t, err)
		require.Equal(t, value, got.AmountOut.Dec(), "amountOut")
		require.True(t, got.AmountIn.Eq(c.Amount), "amountIn")
		require.False(t, got.AmountOut.IsZero())
	case kind == "p":
		require.Equal(t, "17", value, "only the arithmetic overflow panic is expected")
		require.ErrorIs(t, err, ErrOverflow)
	default:
		name, known := routerErrors[value]
		require.True(t, known, "unknown router error %s", value)
		switch name {
		case "zero_out":
			require.NoError(t, err)
			require.True(t, got.AmountOut.IsZero())
		case "zero_balance":
			require.ErrorIs(t, err, ErrZeroBalance)
		default:
			require.Error(t, err)
		}
	}
	if c.Site != "" {
		require.Equal(t, c.Site, siteOfProgram(prog, raw[in].ToBig(), raw[out].ToBig(), c.Amount.ToBig(), inLt), "the overflow site")
	}
}

// expectedFees applies the collection rule to the quote with a maker wallet that is not binding: a pull
// succeeds when the strategy still holds the fee after the earlier pulls.
func expectedFees(feeIns []uint256.Int, rawIn *uint256.Int, launch bool) (collected uint256.Int, refused bool) {
	for _, fee := range feeIns {
		if fee.IsZero() {
			continue
		}
		var left uint256.Int
		left.Sub(rawIn, &collected)
		if fee.Gt(&left) {
			if launch {
				return collected, true
			}
			continue
		}
		collected.Add(&collected, &fee)
	}
	return collected, false
}

func TestCaseMatrix(t *testing.T) {
	m := buildMatrix()
	oracle := loadOracle(t)
	require.Equal(t, m.digest(), oracle.Digest, "the cell list changed: regenerate testdata/matrix_oracle.json with TestMatrixOracle")
	require.Len(t, oracle.Results, len(m.Cells))

	for i, c := range m.Cells {
		s := m.Strategies[c.Strategy]
		t.Run(matrixTestName(c), func(t *testing.T) {
			order := m.order(c.Strategy)
			program, err := ParseOrder(order, s.Launch)
			if s.WantParse != "" {
				requireReason(t, err, s.WantParse, c.Id)
				return
			}
			require.NoError(t, err)
			checkAgainstRouter(t, program, c, s.Raw, oracle.Results[i])

			// The simulator on the same state, wallet not binding: caps and fee collection follow the quote.
			if s.Unshippable {
				return
			}
			in, out := c.In, 1-c.In
			res, qerr := runQuote(program, s.Raw[in], s.Raw[out], c.Amount, in == 0)
			sim := newSim(t, poolOpts{program: s.Program, app: m.app(c.Strategy), maker: m.maker(c.Strategy), raw: s.Raw,
				tokens: [2]string{addrHex(matrixToken0), addrHex(matrixToken1)}})
			got, err := quoteOut(sim, in, c.Amount, nil)
			switch {
			case qerr != nil:
				require.ErrorIs(t, err, qerr)
			case res.AmountOut.IsZero():
				require.ErrorIs(t, err, ErrZeroAmountOut)
			case res.AmountOut.Gt(s.Raw[out]):
				require.ErrorIs(t, err, ErrInsufficientRaw)
			default:
				collected, refused := expectedFees(res.FeeIns, s.Raw[in], s.Launch)
				var after uint256.Int
				after.Sub(s.Raw[in], &collected)
				if _, overflow := after.AddOverflow(&after, c.Amount); overflow || after.Gt(uMaxUint248) {
					require.ErrorIs(t, err, ErrOverflow)
					return
				}
				if refused {
					require.ErrorIs(t, err, ErrFeeShortfall)
					return
				}
				require.NoError(t, err)
				require.Equal(t, res.AmountOut.Dec(), got.TokenAmountOut.Amount.String())
				require.Equal(t, collected.Dec(), got.Fee.Amount.String())
			}
		})
	}
}

// Every oracle cell is accounted for: the cells shipped as strategies the router answered, the ones the
// interpreter refuses, and the unshippable ones.
func TestCaseMatrixAccounting(t *testing.T) {
	m := buildMatrix()
	oracle := loadOracle(t)
	require.Equal(t, m.digest(), oracle.Digest)
	var quoted, reverted, unshippable, refused int
	groups := map[string]int{}
	for i, c := range m.Cells {
		groups[m.Strategies[c.Strategy].Group]++
		switch {
		case m.Strategies[c.Strategy].WantParse != "":
			refused++
		case oracle.Results[i] == "-":
			unshippable++
		case strings.HasPrefix(oracle.Results[i], "o:"):
			quoted++
		default:
			reverted++
		}
	}
	t.Logf("matrix: %d strategies, %d cells: %d quoted by the router, %d router reverts, %d refused programs, %d unshippable; groups %v",
		len(m.Strategies), len(m.Cells), quoted, reverted, refused, unshippable, groups)
	require.Greater(t, quoted, 1000)
	require.Greater(t, reverted, 100)
	require.Positive(t, unshippable)
	// Every router error the oracle recorded is one the matrix names.
	for _, r := range oracle.Results {
		if after, ok := strings.CutPrefix(r, "r:"); ok {
			_, known := routerErrors[after]
			require.True(t, known, r)
		}
	}
}

// TestWriteMatrixDoc writes the case matrix as a markdown table (ONEINCH_AQUA_MATRIX_DOC=<path>).
func TestWriteMatrixDoc(t *testing.T) {
	path := os.Getenv("ONEINCH_AQUA_MATRIX_DOC")
	if path == "" {
		t.Skip("set ONEINCH_AQUA_MATRIX_DOC")
	}
	m := buildMatrix()
	oracle := loadOracle(t)
	var b strings.Builder
	b.WriteString("| # | Group | Strategy | Router | Dir | Amount | Router answer | Interpreter expectation | Test |\n|---|---|---|---|---|---|---|---|---|\n")
	for i, c := range m.Cells {
		s := m.Strategies[c.Strategy]
		router := map[bool]string{false: "current", true: "launch"}[s.Launch]
		answer, expect := oracle.Results[i], ""
		switch {
		case s.WantParse != "":
			expect = "refused: " + string(s.WantParse)
		case answer == "-":
			expect = "math/big model (raw above uint248, cannot be shipped)"
		case strings.HasPrefix(answer, "o:"):
			expect = "amountOut " + strings.TrimPrefix(answer, "o:")
		default:
			expect = "same revert: " + routerErrors[strings.TrimPrefix(strings.TrimPrefix(answer, "r:"), "p:")]
			if strings.HasPrefix(answer, "p:") {
				expect = "ErrOverflow at " + c.Site
			}
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %d | %s | %s | %s | `TestCaseMatrix/%s` |\n", i+1, s.Group, s.Name, router, c.In, c.Amount.Dec(), answer, expect, matrixTestName(c))
	}
	require.NoError(t, os.WriteFile(path, []byte(b.String()), 0o644))
}

// The exact-out cells of the case matrix: a strategy, a direction and an amountOut to buy. They come from the
// same strategies as the exact-in cells. The targets are the edges (1, 2), and for every exact-in cell the
// router quoted, its amountOut X with X-1 and X+1: so the amountIn of each is at most the cell's amountIn
// (the router quoted X for it) and sits on the rounding steps of the program.

type exactOutCell struct {
	Id       string
	Strategy int
	In       int
	Target   *uint256.Int
	// Quoted is the exact-in cell whose router answer the target comes from (-1 for the edges).
	Quoted int
}

func (m *matrix) exactOutCells(t testing.TB) []exactOutCell {
	t.Helper()
	oracle := loadOracle(t)
	require.Equal(t, m.digest(), oracle.Digest)
	var cells []exactOutCell
	seen := map[string]bool{}
	add := func(s, in int, target *uint256.Int, quoted int) {
		if target.IsZero() {
			return
		}
		id := fmt.Sprintf("%s/%s/dir%d/out%s", m.Strategies[s].Group, m.Strategies[s].Name, in, target.Dec())
		if seen[id] {
			return
		}
		seen[id] = true
		cells = append(cells, exactOutCell{Id: id, Strategy: s, In: in, Target: target, Quoted: quoted})
	}
	eligible := func(s int) bool { return m.Strategies[s].WantParse == "" && !m.Strategies[s].Unshippable }
	for s := range m.Strategies {
		if !eligible(s) {
			continue
		}
		for in := range 2 {
			add(s, in, uint256.NewInt(1), -1)
			add(s, in, uint256.NewInt(2), -1)
			add(s, in, new(uint256.Int).Rsh(m.Strategies[s].Raw[1-in], 1), -1)
		}
	}
	for i, c := range m.Cells {
		if !eligible(c.Strategy) || !strings.HasPrefix(oracle.Results[i], "o:") {
			continue
		}
		x := u(strings.TrimPrefix(oracle.Results[i], "o:"))
		add(c.Strategy, c.In, x, i)
		add(c.Strategy, c.In, new(uint256.Int).AddUint64(x, 1), i)
		if x.Gt(uint256.NewInt(1)) {
			add(c.Strategy, c.In, new(uint256.Int).SubUint64(x, 1), i)
		}
	}
	return cells
}

func (m *matrix) simFor(t testing.TB, s int) *PoolSimulator {
	st := m.Strategies[s]
	return newSim(t, poolOpts{program: st.Program, app: m.app(s), maker: m.maker(s), raw: st.Raw,
		wallet: openWallet, allowance: openWallet, tokens: [2]string{addrHex(matrixToken0), addrHex(matrixToken1)}})
}

// Every exact-out cell offline: CalcAmountIn is the smallest amountIn (checked against the math/big model),
// and where the router quoted X for an amountIn, the smallest amountIn for X is not above it.
func TestCaseMatrixExactOut(t *testing.T) {
	t.Parallel()
	m := buildMatrix()
	cells := m.exactOutCells(t)
	oracle := loadOracle(t)
	require.Greater(t, len(cells), 3000)
	sims := map[int]*PoolSimulator{}
	var ok, refused, bounded int
	for _, c := range cells {
		s, found := sims[c.Strategy]
		if !found {
			s = m.simFor(t, c.Strategy)
			sims[c.Strategy] = s
		}
		res, err := verifyAmountIn(t, s, c.In, c.Target, nil)
		if err != nil {
			refused++
		} else {
			ok++
		}
		if c.Quoted < 0 {
			continue
		}
		cell := m.Cells[c.Quoted]
		x := u(strings.TrimPrefix(oracle.Results[c.Quoted], "o:"))
		if !c.Target.Eq(x) {
			continue
		}
		// the router quoted x for cell.Amount, and the simulator agrees on that quote unless a cap refuses it
		if _, qerr := quoteOut(s, cell.In, cell.Amount, nil); qerr != nil {
			continue
		}
		require.NoError(t, err, c.Id)
		require.LessOrEqual(t, res.TokenAmountIn.Amount.Cmp(cell.Amount.ToBig()), 0, "%s: more than the amountIn the router quoted it for", c.Id)
		bounded++
	}
	t.Logf("matrix exact-out: %d cells, %d with an amountIn, %d refused, %d bounded by a router-quoted amountIn", len(cells), ok, refused, bounded)
	require.Greater(t, ok, 2000)
	require.Greater(t, refused, 100)
	require.Greater(t, bounded, 500)
}

// exactOutVector is one exact-out cell with the amountIn of the simulator, for the router replay.
type exactOutVector struct {
	Id       string `json:"id"`
	Strategy int    `json:"strategy"`
	In       int    `json:"in"`
	Target   string `json:"target"`
	AmountIn string `json:"amountIn"`
}

func (m *matrix) exactOutVectors(t testing.TB) []exactOutVector {
	var out []exactOutVector
	sims := map[int]*PoolSimulator{}
	for _, c := range m.exactOutCells(t) {
		s, ok := sims[c.Strategy]
		if !ok {
			s = m.simFor(t, c.Strategy)
			sims[c.Strategy] = s
		}
		res, err := s.CalcAmountIn(inParams(s, c.In, c.Target, nil))
		if err != nil {
			continue
		}
		out = append(out, exactOutVector{Id: c.Id, Strategy: c.Strategy, In: c.In, Target: c.Target.Dec(), AmountIn: res.TokenAmountIn.Amount.String()})
	}
	return out
}

// TestMatrixExactOutEmit writes the exact-out vectors (ONEINCH_AQUA_EXACTOUT_VECTORS=<path>) for review or
// for a router replay by other tooling.
func TestMatrixExactOutEmit(t *testing.T) {
	path := os.Getenv("ONEINCH_AQUA_EXACTOUT_VECTORS")
	if path == "" {
		t.Skip("set ONEINCH_AQUA_EXACTOUT_VECTORS")
	}
	b, err := json.Marshal(buildMatrix().exactOutVectors(t))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0o644))
}

// oracleOut returns the router's amountOut for a matrix cell id.
func oracleOut(t *testing.T, id string) *uint256.Int {
	t.Helper()
	m := buildMatrix()
	oracle := loadOracle(t)
	for i, c := range m.Cells {
		if c.Id == id {
			require.True(t, strings.HasPrefix(oracle.Results[i], "o:"), id)
			return u(strings.TrimPrefix(oracle.Results[i], "o:"))
		}
	}
	t.Fatalf("no matrix cell %s", id)
	return nil
}

const capCellId = "router/concentrate-flat-xyc-current-A/dir0/amt1000000000000000000"

// Caps: the maker wallet limit binding, the strategy balance binding, both, or neither. X is the router's
// amountOut for the amount; every case is a cap placed relative to X.
func TestCaseMatrixCaps(t *testing.T) {
	t.Parallel()
	x := oracleOut(t, capCellId)
	xm1, xp1 := new(uint256.Int).SubUint64(x, 1), new(uint256.Int).AddUint64(x, 1)
	program := buildProgram("CF", "none")
	amount := e18(1)
	huge := e18(1_000_000_000)

	// Own snapshot (no SwapLimit): min(balance, allowance).
	for name, tc := range map[string]struct {
		wallet, allowance *uint256.Int
		want              error
	}{
		"no cap":                         {huge, unlimited, nil},
		"balance equals out":             {x, unlimited, nil},
		"balance one below out":          {xm1, unlimited, ErrInsufficientWallet},
		"balance one above out":          {xp1, unlimited, nil},
		"balance zero":                   {new(uint256.Int), unlimited, ErrInsufficientWallet},
		"allowance equals out":           {huge, x, nil},
		"allowance one below out":        {huge, xm1, ErrInsufficientWallet},
		"allowance zero":                 {huge, new(uint256.Int), ErrInsufficientWallet},
		"balance below, allowance above": {xm1, huge, ErrInsufficientWallet},
		"balance above, allowance below": {huge, xm1, ErrInsufficientWallet},
	} {
		s := newSim(t, poolOpts{program: program, raw: profileA.Raw, wallet: [2]*uint256.Int{nil, tc.wallet}, allowance: [2]*uint256.Int{nil, tc.allowance}})
		res, err := quoteOut(s, 0, amount, nil)
		if tc.want != nil {
			require.ErrorIs(t, err, tc.want, name)
			continue
		}
		require.NoError(t, err, name)
		require.Equal(t, x.Dec(), res.TokenAmountOut.Amount.String(), name)
	}

	// Shared limit: the key replaces the own snapshot.
	s := newSim(t, poolOpts{program: program, raw: profileA.Raw})
	key := s.limitKeys[1]
	limit := func(v *big.Int) *swaplimit.Inventory {
		m := map[string]*big.Int{}
		if v != nil {
			m[key] = v
		}
		return swaplimit.NewInventory(DexType, m)
	}
	for name, tc := range map[string]struct {
		limit *big.Int
		want  error
	}{
		"limit equals out":    {x.ToBig(), nil},
		"limit one below out": {xm1.ToBig(), ErrInsufficientWallet},
		"limit one above out": {xp1.ToBig(), nil},
		"limit zero":          {new(big.Int), ErrInsufficientWallet},
		"missing key":         {nil, ErrInsufficientWallet},
	} {
		res, err := quoteOut(s, 0, amount, limit(tc.limit))
		if tc.want != nil {
			require.ErrorIs(t, err, tc.want, name)
			continue
		}
		require.NoError(t, err, name)
		require.Equal(t, x.Dec(), res.TokenAmountOut.Amount.String(), name)
	}

	// Which cap binds: the strategy balance or the wallet. The rawcap cells straddle the out balance.
	raw := [2]*uint256.Int{e18(1), e18(1)}
	m := buildMatrix()
	oracle := loadOracle(t)
	var below, above int
	for i, c := range m.Cells {
		if m.Strategies[c.Strategy].Group != "rawcap" {
			continue
		}
		out := u(strings.TrimPrefix(oracle.Results[i], "o:"))
		sim := newSim(t, poolOpts{program: m.Strategies[c.Strategy].Program, raw: raw})
		for name, tc := range map[string]struct {
			wallet *uint256.Int
			want   error
		}{"wallet not binding": {huge, nil}, "wallet binding": {new(uint256.Int).SubUint64(out, 1), ErrInsufficientWallet}} {
			sim.State.Wallet[1] = *tc.wallet
			_, err := quoteOut(sim, 0, c.Amount, nil)
			switch {
			case out.Gt(raw[1]):
				require.ErrorIs(t, err, ErrInsufficientRaw, "%s %s: the strategy balance binds first", c.Id, name)
			case tc.want != nil:
				require.ErrorIs(t, err, tc.want, "%s %s", c.Id, name)
			default:
				require.NoError(t, err, "%s %s", c.Id, name)
			}
		}
		if out.Gt(raw[1]) {
			above++
		} else {
			below++
		}
	}
	require.Positive(t, below, "an amount whose out is within the strategy balance")
	require.Positive(t, above, "an amount whose out exceeds the strategy balance")
}

// State guards: every state the tracker can write, plus the two the simulator refuses by itself.
func TestCaseMatrixStateGuards(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		o    poolOpts
		want error
	}{
		"live":                  {poolOpts{}, nil},
		"docked":                {poolOpts{docked: true}, ErrStrategyTerminal},
		"inactive":              {poolOpts{inactive: true}, ErrStrategyTerminal},
		"docked and inactive":   {poolOpts{docked: true, inactive: true}, ErrStrategyTerminal},
		"no wallet snapshot":    {poolOpts{rawBlock: 5, walletBlk: 0}, ErrNoWalletSnapshot},
		"wallet snapshot only":  {poolOpts{rawBlock: 0, walletBlk: 5}, ErrStateMixedBlock},
		"raw older than wallet": {poolOpts{rawBlock: 4, walletBlk: 5}, ErrStateMixedBlock},
		"raw newer than wallet": {poolOpts{rawBlock: 6, walletBlk: 5}, ErrStateMixedBlock},
		"same block":            {poolOpts{rawBlock: 5, walletBlk: 5}, nil},
	} {
		for in := range 2 {
			s := newSim(t, tc.o)
			_, err := quoteOut(s, in, e18(1), nil)
			if tc.want == nil {
				require.NoError(t, err, name)
				continue
			}
			require.ErrorIs(t, err, tc.want, name)
			if tc.want != ErrStateMixedBlock { // a mixed state never comes from the tracker, and publishes its snapshot
				require.Nil(t, s.CalculateLimit(), name+": no limit keys")
			}
		}
	}

	// tokensCount as the tracker maps it.
	counts := map[int64]string{0: "inactive", 1: "inactive", 2: "live", 3: "inactive", 0xfe: "inactive", tokensCountDocked: "docked"}
	for count, want := range counts {
		chain := newFakeChain()
		e := listedPool(t, fmt.Sprint(count), poolOpts{})
		chain.setStrategy(poolOpts{}, e, count, e18(10), e18(10))
		chain.setWallet(testMaker.Hex(), testToken0, e18(5), unlimited)
		chain.setWallet(testMaker.Hex(), testToken1, e18(5), unlimited)
		tr, _ := chain.tracker(t)
		pools, errs := tr.GetNewPoolStates(t.Context(), []entity.Pool{e})
		require.NoError(t, errs[0])
		x := decodeExtra(t, pools[0])
		sim, err := NewPoolSimulator(pools[0])
		require.NoError(t, err)
		_, qerr := quoteOut(sim, 0, e18(1), nil)
		switch want {
		case "live":
			require.False(t, x.Docked || x.Inactive, count)
			require.NoError(t, qerr, count)
		case "docked":
			require.True(t, x.Docked, count)
			require.ErrorIs(t, qerr, ErrStrategyTerminal)
		default:
			require.True(t, x.Inactive && !x.Docked, count)
			require.ErrorIs(t, qerr, ErrStrategyTerminal)
		}
	}
}

// snapshot copies everything a quote could change.
type simSnapshot struct {
	state    state
	reserves []string
	limit    map[string]string
}

func snapshotOf(s *PoolSimulator, inv *swaplimit.Inventory, keys ...string) simSnapshot {
	snap := simSnapshot{state: s.State, limit: map[string]string{}}
	for _, r := range s.Info.Reserves {
		snap.reserves = append(snap.reserves, r.String())
	}
	if inv != nil {
		for _, k := range keys {
			if v := inv.GetLimit(k); v != nil {
				snap.limit[k] = v.String()
			}
		}
	}
	return snap
}

// Determinism and purity: the same quote repeated gives the same result, and a quote leaves the pool and the
// shared limit as they were, on success and on every error.
func TestSequencesDeterminismAndPurity(t *testing.T) {
	t.Parallel()
	feeProgram := buildProgram("PCF", "trail")
	s := newSim(t, poolOpts{program: feeProgram})
	inv := swaplimit.NewInventory(DexType, map[string]*big.Int{s.limitKeys[0]: e18(500).ToBig(), s.limitKeys[1]: e18(900).ToBig()})
	before := snapshotOf(s, inv, s.limitKeys[0], s.limitKeys[1])
	first, err := quoteOut(s, 0, e18(10), inv)
	require.NoError(t, err)
	for range 200 {
		res, err := quoteOut(s, 0, e18(10), inv)
		require.NoError(t, err)
		require.Equal(t, first.TokenAmountOut.Amount, res.TokenAmountOut.Amount)
		require.Equal(t, first.Fee.Amount, res.Fee.Amount)
		require.Equal(t, first.Gas, res.Gas)
		require.Equal(t, first.SwapInfo, res.SwapInfo)
	}
	require.Equal(t, before, snapshotOf(s, inv, s.limitKeys[0], s.limitKeys[1]), "success changed nothing")

	// A result does not alias the state: changing it changes nothing in the pool.
	first.TokenAmountOut.Amount.SetInt64(7)
	again, err := quoteOut(s, 0, e18(10), inv)
	require.NoError(t, err)
	require.NotEqual(t, "7", again.TokenAmountOut.Amount.String())

	// Every error path leaves the state as it was.
	cases := map[string]func() error{
		"zero amount out": func() error { _, err := quoteOut(s, 0, uint256.NewInt(1), inv); return err },
		"unknown token": func() error {
			_, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: "0xno", Amount: big.NewInt(1)}, TokenOut: s.Info.Tokens[1]})
			return err
		},
		"amount too big": func() error { _, err := quoteOut(s, 0, e18(100_000_000), inv); return err },
		"limit missing": func() error {
			_, err := quoteOut(s, 0, e18(10), swaplimit.NewInventory(DexType, map[string]*big.Int{}))
			return err
		},
	}
	for name, f := range cases {
		require.Error(t, f(), name)
		require.Equal(t, before, snapshotOf(s, inv, s.limitKeys[0], s.limitKeys[1]), name)
		res, err := quoteOut(s, 0, e18(10), inv)
		require.NoError(t, err, name)
		require.Equal(t, first.Gas, res.Gas)
	}
}

// freshFrom builds a simulator from the state of another: what a refresh would store after the swaps.
func freshFrom(t *testing.T, s *PoolSimulator, program []byte, tokens [2]string) *PoolSimulator {
	t.Helper()
	return newSim(t, poolOpts{program: program, tokens: tokens,
		raw: [2]*uint256.Int{&s.State.Raw[0], &s.State.Raw[1]}, wallet: [2]*uint256.Int{&s.State.Wallet[0], &s.State.Wallet[1]},
		allowance: [2]*uint256.Int{&s.State.Allowance[0], &s.State.Allowance[1]}})
}

// Multi-hop on one pool: after each UpdateBalance the next quote equals a quote on a pool built from the
// post-state.
func TestSequencesMultiHopSamePool(t *testing.T) {
	t.Parallel()
	for _, ls := range liveShapes {
		program := buildProgram(ls.wrappers, ls.salt)
		s := newSim(t, poolOpts{program: program, wallet: [2]*uint256.Int{e18(900), e18(900)}, allowance: [2]*uint256.Int{e18(5000), unlimited}})
		tokens := [2]string{s.Info.Tokens[0], s.Info.Tokens[1]}
		for hop, step := range []struct {
			in     int
			amount *uint256.Int
		}{{0, e18(10)}, {1, e18(7)}, {0, e18(3)}, {0, e18(25)}, {1, e18(1)}, {1, e18(40)}, {0, e18(2)}} {
			fresh := freshFrom(t, s, program, tokens)
			want, wantErr := quoteOut(fresh, step.in, step.amount, nil)
			got, err := quoteOut(s, step.in, step.amount, nil)
			if wantErr != nil {
				require.ErrorIs(t, err, wantErr, "%s hop %d", ls.name, hop)
				continue
			}
			require.NoError(t, err, "%s hop %d", ls.name, hop)
			require.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount, "%s hop %d", ls.name, hop)
			require.Equal(t, want.Fee.Amount, got.Fee.Amount, "%s hop %d", ls.name, hop)
			s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: tokens[step.in], Amount: step.amount.ToBig()},
				TokenAmountOut: *got.TokenAmountOut, Fee: *got.Fee, SwapInfo: got.SwapInfo})
		}
	}
}

// Two strategies of one maker in one route share the wallet limit under the min merge.
func TestSequencesTwoPoolsSharedLimit(t *testing.T) {
	t.Parallel()
	program := buildProgram("CF", "none")
	a := newSim(t, poolOpts{strategyId: "a", program: program, wallet: [2]*uint256.Int{e18(500), e18(900)}})
	b := newSim(t, poolOpts{strategyId: "b", program: program, wallet: [2]*uint256.Int{e18(400), e18(700)}})
	merged := map[string]*big.Int{}
	pool.MergeSwapLimits(merged, a)
	pool.MergeSwapLimits(merged, b)
	key0, key1 := a.limitKeys[0], a.limitKeys[1]
	require.Equal(t, e18(400).Dec(), merged[key0].String(), "min of the snapshots")
	require.Equal(t, e18(700).Dec(), merged[key1].String())

	outA, err := quoteOut(a, 0, e18(10), nil)
	require.NoError(t, err)
	outB, err := quoteOut(b, 0, e18(10), nil)
	require.NoError(t, err)
	need := new(big.Int).Add(outA.TokenAmountOut.Amount, outB.TokenAmountOut.Amount)
	for name, tc := range map[string]struct {
		limit  *big.Int
		wantB2 error
	}{
		"room for both hops":      {new(big.Int).Set(need), nil},
		"one wei short of both":   {new(big.Int).Sub(need, big.NewInt(1)), ErrInsufficientWallet},
		"room for the first only": {new(big.Int).Set(outA.TokenAmountOut.Amount), ErrInsufficientWallet},
	} {
		inv := swaplimit.NewInventory(DexType, map[string]*big.Int{key0: e18(400).ToBig(), key1: tc.limit})
		simA, simB := a.CloneState().(*PoolSimulator), b.CloneState().(*PoolSimulator)
		first, err := swapOnce(simA, 0, e18(10), inv)
		require.NoError(t, err, name)
		require.Equal(t, outA.TokenAmountOut.Amount, first.TokenAmountOut.Amount)
		require.Equal(t, new(big.Int).Sub(tc.limit, first.TokenAmountOut.Amount).String(), inv.GetLimit(key1).String(), name)
		// The second hop equals a quote against the limit that is left, and the other strategy's own state is untouched.
		second, err := quoteOut(simB, 0, e18(10), inv)
		if tc.wantB2 != nil {
			require.ErrorIs(t, err, tc.wantB2, name)
		} else {
			require.NoError(t, err, name)
			require.Equal(t, outB.TokenAmountOut.Amount, second.TokenAmountOut.Amount, name)
		}
		require.Equal(t, e18(1000).Dec(), simB.State.Raw[0].Dec())
	}
	// The original pools were never touched by the cloned runs.
	require.Equal(t, e18(1000).Dec(), a.State.Raw[0].Dec())
}

// The tokenIn credit depends on the allowance: with an unlimited one the taker's push raises the wallet
// limit and the reverse hop can draw on it; with a finite one nothing is credited.
func TestSequencesConditionalCredit(t *testing.T) {
	t.Parallel()
	program := prog(iFlat(10_000_000), iSwap())
	for name, tc := range map[string]struct {
		allowance0 *uint256.Int
		credited   bool
	}{
		"unlimited":               {unlimited, true},
		"at the 2^128 threshold":  {new(uint256.Int).Lsh(uint256.NewInt(1), 128), true},
		"one below the threshold": {new(uint256.Int).SubUint64(new(uint256.Int).Lsh(uint256.NewInt(1), 128), 1), false},
		"finite and small":        {e18(50), false},
	} {
		s := newSim(t, poolOpts{program: program, allowance: [2]*uint256.Int{tc.allowance0, nil}})
		inv := swaplimit.NewInventory(DexType, map[string]*big.Int{s.limitKeys[0]: new(big.Int), s.limitKeys[1]: e18(900).ToBig()})
		_, err := swapOnce(s, 0, e18(100), inv)
		require.NoError(t, err, name)
		require.Equal(t, tc.credited, inv.GetLimit(s.limitKeys[0]).Sign() > 0, name)
		// The reverse hop sells token1 for token0: it needs limit on token0.
		_, err = quoteOut(s, 1, e18(1), inv)
		if tc.credited {
			require.NoError(t, err, name)
		} else {
			require.ErrorIs(t, err, ErrInsufficientWallet, name)
		}
	}
}

// CloneState isolation, both ways.
func TestSequencesCloneIsolation(t *testing.T) {
	t.Parallel()
	program := buildProgram("PCF", "trail")
	original := newSim(t, poolOpts{program: program, wallet: [2]*uint256.Int{e18(900), e18(900)}, allowance: [2]*uint256.Int{e18(5000), e18(5000)}})
	clone := original.CloneState().(*PoolSimulator)
	frozen := *original
	frozenClone := *clone

	// The clone swaps: the original keeps its state and its quote.
	_, err := swapOnce(clone, 0, e18(10), nil)
	require.NoError(t, err)
	require.Equal(t, frozen.State, original.State)
	require.Equal(t, frozen.Info.Reserves, original.Info.Reserves)
	a, err := quoteOut(original, 0, e18(10), nil)
	require.NoError(t, err)
	b, err := quoteOut(newSim(t, poolOpts{program: program, wallet: [2]*uint256.Int{e18(900), e18(900)}, allowance: [2]*uint256.Int{e18(5000), e18(5000)}}), 0, e18(10), nil)
	require.NoError(t, err)
	require.Equal(t, b.TokenAmountOut.Amount, a.TokenAmountOut.Amount)

	// The original swaps: the clone, taken before, keeps its own state.
	cloneState := clone.State
	_, err = swapOnce(original, 1, e18(5), nil)
	require.NoError(t, err)
	require.Equal(t, cloneState, clone.State)
	require.NotEqual(t, frozenClone.State, clone.State, "the clone moved once, by its own swap")

	// The reserves slice is replaced, not edited, so a clone made earlier keeps its reserves.
	third := original.CloneState().(*PoolSimulator)
	reservesBefore := []string{third.Info.Reserves[0].String(), third.Info.Reserves[1].String()}
	_, err = swapOnce(original, 0, e18(3), nil)
	require.NoError(t, err)
	require.Equal(t, reservesBefore, []string{third.Info.Reserves[0].String(), third.Info.Reserves[1].String()})
	_ = common.Address{}
}

// The rounding cells are not accidents: for the flat and protocol fee cells most amounts leave a remainder,
// so the ceiling of the flat fee and the floor of the protocol fee give different amounts, and the router
// agreed with the interpreter on each of them (TestCaseMatrix).
func TestRoundingCellsHitRemainders(t *testing.T) {
	t.Parallel()
	m := buildMatrix()
	var flat, protocol int
	for _, c := range m.Cells {
		s := m.Strategies[c.Strategy]
		if s.Group != "rounding" {
			continue
		}
		bps := map[string]uint64{"flat-ceil": 10_000_000, "flat-ceil-bps1": 1, "protocol-floor": 10_000_000}[s.Name]
		if bps == 0 {
			continue
		}
		prod := new(big.Int).Mul(c.Amount.ToBig(), new(big.Int).SetUint64(bps))
		if new(big.Int).Mod(prod, big.NewInt(feeBpsDenominator)).Sign() == 0 {
			continue
		}
		if s.Name == "protocol-floor" {
			protocol++
		} else {
			flat++
		}
	}
	require.GreaterOrEqual(t, flat, 8, "flat fee cells whose fee rounds up")
	require.GreaterOrEqual(t, protocol, 4, "protocol fee cells whose fee rounds down")
}
