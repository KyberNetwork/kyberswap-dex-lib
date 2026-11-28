package efficientprop

import (
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestConfigValidate(t *testing.T) {
	cfg := Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{{
		Address:         "0x0000000000403AEB66691063CC2FD42E71D6ACD6",
		DoorFactory:     "0x39ED9993449C96503F6A629CCD2BB5F132D7CC93",
		FillMode:        FillModeFront,
		Implementations: []string{"0x6000000000000000000000000000000000000001"},
		BaseTime:        "0x4200000000000000000000000000000000000030",
	}}}
	require.NoError(t, cfg.validate())
	assert.Equal(t, "0x0000000000403aeb66691063cc2fd42e71d6acd6", cfg.Fronts[0].Address)
	assert.Equal(t, "0x39ed9993449c96503f6a629ccd2bb5f132d7cc93", cfg.Fronts[0].DoorFactory)
	assert.Equal(t, defaultMaxListings, cfg.maxListings())
	assert.Equal(t, DexType, cfg.dexID())

	for name, bad := range map[string]Config{
		"no chain":   {Fronts: cfg.Fronts},
		"no fronts":  {ChainID: valueobject.ChainIDBase},
		"no factory": {ChainID: valueobject.ChainIDBase, Fronts: []Front{{Address: cfg.Fronts[0].Address}}},
		"zero front": {ChainID: valueobject.ChainIDBase, Fronts: []Front{{
			Address: "0x0000000000000000000000000000000000000000", DoorFactory: cfg.Fronts[0].DoorFactory}}},
		"malformed": {ChainID: valueobject.ChainIDBase, Fronts: []Front{{Address: "front",
			DoorFactory: cfg.Fronts[0].DoorFactory}}},
		"front twice":       {ChainID: valueobject.ChainIDBase, Fronts: []Front{cfg.Fronts[0], cfg.Fronts[0]}},
		"negative scans":    {ChainID: valueobject.ChainIDBase, Fronts: cfg.Fronts, MaxListings: -1},
		"no fill mode":      {ChainID: valueobject.ChainIDBase, Fronts: []Front{withMode(cfg.Fronts[0], "")}},
		"unknown fill mode": {ChainID: valueobject.ChainIDBase, Fronts: []Front{withMode(cfg.Fronts[0], "pool")}},
		"fill mode in caps": {ChainID: valueobject.ChainIDBase, Fronts: []Front{withMode(cfg.Fronts[0], "Front")}},
		"malformed source":  {ChainID: valueobject.ChainIDBase, Fronts: []Front{withSources(cfg.Fronts[0], "lane")}},
		"zero source": {ChainID: valueobject.ChainIDBase, Fronts: []Front{withSources(cfg.Fronts[0],
			"0x0000000000000000000000000000000000000000")}},
		"no implementation":        {ChainID: valueobject.ChainIDBase, Fronts: []Front{withImpls(cfg.Fronts[0])}},
		"malformed implementation": {ChainID: valueobject.ChainIDBase, Fronts: []Front{withImpls(cfg.Fronts[0], "impl")}},
		"zero implementation": {ChainID: valueobject.ChainIDBase, Fronts: []Front{withImpls(cfg.Fronts[0],
			"0x0000000000000000000000000000000000000000")}},
		"malformed baseTime": {ChainID: valueobject.ChainIDBase, Fronts: []Front{withBaseTime(cfg.Fronts[0], "clock")}},
		"zero baseTime": {ChainID: valueobject.ChainIDBase, Fronts: []Front{withBaseTime(cfg.Fronts[0],
			"0x0000000000000000000000000000000000000000")}},
	} {
		assert.ErrorIs(t, bad.validate(), ErrInvalidConfig, name)
	}
}

// withBaseTime is f with its chain's millisecond-time address set to bt.
func withBaseTime(f Front, bt string) Front {
	f.BaseTime = bt
	return f
}

// A front's millisecond-time address is optional where its clock reads no block count, and lowercased where set.
func TestConfigBaseTime(t *testing.T) {
	cfg := Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{withBaseTime(testConfig().Fronts[0], "")}}
	require.NoError(t, cfg.validate(), "not required by the config alone")
	cfg.Fronts[0].BaseTime = "0x420000000000000000000000000000000000003A"
	require.NoError(t, cfg.validate())
	assert.Equal(t, "0x420000000000000000000000000000000000003a", cfg.Fronts[0].BaseTime)
}

func withSources(f Front, srcs ...string) Front {
	f.ApprovedSources = srcs
	return f
}

// A front's approved price sources are optional, none by default; each is a nonzero address, kept lowercased, and the
// front approves exactly those.
func TestConfigApprovedSources(t *testing.T) {
	cfg := Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{{
		Address:         "0x0000000000403AEB66691063CC2FD42E71D6ACD6",
		DoorFactory:     "0x39ED9993449C96503F6A629CCD2BB5F132D7CC93",
		FillMode:        FillModeFront,
		Implementations: []string{"0x6000000000000000000000000000000000000001"},
	}}}
	require.NoError(t, cfg.validate())
	assert.Empty(t, cfg.Fronts[0].ApprovedSources, "none by default")
	assert.False(t, cfg.Fronts[0].approves(common.HexToAddress("0x5000000000000000000000000000000000000aBc")))

	cfg.Fronts[0].ApprovedSources = []string{"0x5000000000000000000000000000000000000aBc"}
	require.NoError(t, cfg.validate())
	assert.Equal(t, []string{"0x5000000000000000000000000000000000000abc"}, cfg.Fronts[0].ApprovedSources)
	assert.True(t, cfg.Fronts[0].approves(common.HexToAddress("0x5000000000000000000000000000000000000ABC")))
	assert.False(t, cfg.Fronts[0].approves(common.HexToAddress("0x5000000000000000000000000000000000000def")))
	assert.False(t, cfg.Fronts[0].approves(common.Address{}), "the zero address is no source")
}

func withImpls(f Front, impls ...string) Front {
	f.Implementations = impls
	return f
}

// A front's implementations are required, with no default: the source reads the front's storage by its layout and
// mirrors its clock and pricing rules, so it quotes a front only while its proxy runs a listed implementation. Each is
// a nonzero address, kept lowercased.
func TestConfigImplementations(t *testing.T) {
	cfg := Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{{
		Address:         "0x0000000000403AEB66691063CC2FD42E71D6ACD6",
		DoorFactory:     "0x39ED9993449C96503F6A629CCD2BB5F132D7CC93",
		FillMode:        FillModeFront,
		Implementations: []string{"0x6000000000000000000000000000000000000aBc", "0x6000000000000000000000000000000000000DEF"},
	}}}
	require.NoError(t, cfg.validate())
	assert.Equal(t, []string{"0x6000000000000000000000000000000000000abc", "0x6000000000000000000000000000000000000def"},
		cfg.Fronts[0].Implementations)
	assert.True(t, cfg.Fronts[0].implements(common.HexToAddress("0x6000000000000000000000000000000000000ABC")))
	assert.True(t, cfg.Fronts[0].implements(common.HexToAddress("0x6000000000000000000000000000000000000def")))
	assert.False(t, cfg.Fronts[0].implements(common.HexToAddress("0x6000000000000000000000000000000000000123")))
	assert.False(t, cfg.Fronts[0].implements(common.Address{}))
}

func withMode(f Front, mode string) Front {
	f.FillMode = mode
	return f
}

// Each front names its fill mode, with no default: the adapter's mode word follows it.
func TestConfigFillMode(t *testing.T) {
	for mode, want := range map[string]uint8{FillModeDoor: modeDoor, FillModeFront: modeFront} {
		f := Front{FillMode: mode}
		got, ok := f.mode()
		assert.True(t, ok, mode)
		assert.Equal(t, want, got, mode)
	}
	assert.Equal(t, uint8(0), modeDoor, "the adapter's mode 0 fills through the door")
	assert.Equal(t, uint8(1), modeFront, "the adapter's mode 1 fills through the front")
	_, ok := Front{}.mode()
	assert.False(t, ok, "no default")
}

// The inclusion allowance defaults to 1 s on a chain whose blocks are under a second apart (Base), where a routed fill
// lands within about a second, and to 3 s on any other; a configured allowance stands on every chain.
func TestConfigInclusionAllowance(t *testing.T) {
	for _, c := range []struct {
		chain      valueobject.ChainID
		configured uint64
		want       uint64
	}{
		{valueobject.ChainIDBase, 0, 1},
		{valueobject.ChainIDEthereum, 0, 3},
		{valueobject.ChainIDBSC, 0, 3},
		{valueobject.ChainIDBase, 3, 3},
		{valueobject.ChainIDEthereum, 1, 1},
	} {
		cfg := Config{ChainID: c.chain, InclusionAllowance: c.configured}
		assert.Equal(t, c.want, cfg.inclusionAllowance(), "chain %d, configured %d", c.chain, c.configured)
	}
}

// ethFront is a front on Ethereum: the door's fill mode, and the /state read at urls.
func ethFront(urls ...string) Front {
	return Front{
		Address:         "0x0000000000403AEB66691063CC2FD42E71D6ACD6",
		DoorFactory:     "0x39ED9993449C96503F6A629CCD2BB5F132D7CC93",
		CellCodehash:    "0x9D1E3C1C5F0C7B0F6A2E4D7B8C3A5E1F2B4D6C8A0E9F1A3B5C7D9E0F2A4B6C8D",
		Layout:          LayoutBuilder,
		FillMode:        FillModeDoor,
		Implementations: []string{"0x6000000000000000000000000000000000000001"},
		Titan:           &titan.Config{URLs: urls, Timeout: 2 * time.Second},
	}
}

// On a builder chain (Ethereum) each front names the /state read (its titan block, at least one url) and fills in
// either mode, through the door or through the front's own entries (the fill's Swapped, which a builder matches to
// place the desk's update, is logged by the listing's quote contract in either mode). The titan block is refused on
// any other chain, whose blocks are not the beacon chain's slots.
func TestConfigOnABuilderChain(t *testing.T) {
	cfg := Config{ChainID: valueobject.ChainIDEthereum, Fronts: []Front{ethFront("https://state.example/v1/1/state")}}
	require.NoError(t, cfg.validate())
	assert.True(t, cfg.Fronts[0].builder())
	inFront := Config{ChainID: valueobject.ChainIDEthereum,
		Fronts: []Front{withMode(ethFront("https://state.example/v1/1/state"), FillModeFront)}}
	require.NoError(t, inFront.validate(), "the front's mode on Ethereum")
	m, ok := inFront.Fronts[0].mode()
	assert.True(t, ok)
	assert.Equal(t, modeFront, m)

	noTitan := ethFront()
	noTitan.Titan = nil
	baseTitan := testConfig().Fronts[0]
	baseTitan.Titan = &titan.Config{URLs: []string{"https://state.example/v1/8453/state"}}
	for name, bad := range map[string]Config{
		"no fill mode on Ethereum": {ChainID: valueobject.ChainIDEthereum,
			Fronts: []Front{withMode(ethFront("https://state.example/v1/1/state"), "")}},
		"no titan block on Ethereum": {ChainID: valueobject.ChainIDEthereum, Fronts: []Front{noTitan}},
		"no url":                     {ChainID: valueobject.ChainIDEthereum, Fronts: []Front{ethFront()}},
		"an empty url":               {ChainID: valueobject.ChainIDEthereum, Fronts: []Front{ethFront("")}},
		"a titan block on Base":      {ChainID: valueobject.ChainIDBase, Fronts: []Front{baseTitan}},
	} {
		assert.ErrorIs(t, bad.validate(), ErrInvalidConfig, name)
	}
	assert.False(t, testConfig().Fronts[0].builder(), "Base reads no /state")
}

// A front's /state timeout is 0 (the source's default) or in (0, 10 s]: a longer one holds the front's lock that long
// on a host that hangs.
func TestConfigStateTimeout(t *testing.T) {
	for d, ok := range map[time.Duration]bool{0: true, time.Second: true, maxStateTimeout: true,
		-time.Second: false, maxStateTimeout + time.Nanosecond: false} {
		f := ethFront("https://state.example/v1/1/state")
		f.Titan.Timeout = d
		cfg := Config{ChainID: valueobject.ChainIDEthereum, Fronts: []Front{f}}
		if ok {
			assert.NoError(t, cfg.validate(), "%s", d)
		} else {
			assert.ErrorIs(t, cfg.validate(), ErrInvalidConfig, "%s", d)
		}
	}
	assert.Equal(t, time.Second, defaultStateTimeout)
}

// On the /state read a quote is good for its target block, and route finding stops the inclusion allowance before the
// block's time: 1,000 ms by default on Ethereum and 100 ms on BNB Chain, a configured allowance standing (in seconds
// or in milliseconds, not both). An allowance of a whole block or more would quote no state, and is refused, as is
// one under the chain's least (Ethereum's whole second, as before the allowance took milliseconds; 50 ms on BNB
// Chain); the millisecond allowance is refused on a chain read at the head.
func TestConfigBuilderInclusionAllowance(t *testing.T) {
	for _, c := range []struct {
		chain     valueobject.ChainID
		seconds   uint64
		ms        uint64
		want      uint64
		refused   bool
		titleNote string
	}{
		{chain: valueobject.ChainIDEthereum, want: 1_000, titleNote: "Ethereum's default"},
		{chain: valueobject.ChainIDEthereum, seconds: 2, want: 2_000, titleNote: "seconds on Ethereum"},
		{chain: valueobject.ChainIDEthereum, ms: 1_500, want: 1_500, titleNote: "ms on Ethereum"},
		{chain: valueobject.ChainIDEthereum, ms: 1_000, want: 1_000, titleNote: "Ethereum's least"},
		{chain: valueobject.ChainIDEthereum, ms: 999, refused: true, titleNote: "under Ethereum's second"},
		{chain: valueobject.ChainIDEthereum, seconds: 12, refused: true, titleNote: "a whole slot"},
		{chain: valueobject.ChainIDEthereum, seconds: 1, ms: 1_000, refused: true, titleNote: "both set"},
		{chain: valueobject.ChainIDBSC, want: 100, titleNote: "BNB Chain's default"},
		{chain: valueobject.ChainIDBSC, ms: 200, want: 200, titleNote: "ms on BNB Chain"},
		{chain: valueobject.ChainIDBSC, ms: 50, want: 50, titleNote: "BNB Chain's least"},
		{chain: valueobject.ChainIDBSC, ms: 49, refused: true, titleNote: "under BNB Chain's least"},
		{chain: valueobject.ChainIDBSC, ms: 450, refused: true, titleNote: "a whole BNB block"},
		{chain: valueobject.ChainIDBSC, seconds: 1, refused: true, titleNote: "a second on BNB Chain"},
		{chain: valueobject.ChainIDBase, ms: 100, refused: true, titleNote: "ms on Base"},
	} {
		f := ethFront("https://state.example/v1/1/state")
		switch c.chain {
		case valueobject.ChainIDBase:
			f = testConfig().Fronts[0]
		case valueobject.ChainIDBSC:
			f = bnbFront("https://state.example/v1/56/state")
		}
		cfg := Config{ChainID: c.chain, InclusionAllowance: c.seconds, InclusionAllowanceMs: c.ms, Fronts: []Front{f}}
		if c.refused {
			assert.ErrorIs(t, cfg.validate(), ErrInvalidConfig, c.titleNote)
			continue
		}
		require.NoError(t, cfg.validate(), c.titleNote)
		assert.Equal(t, c.want, cfg.builderAllowanceMs(), c.titleNote)
	}
}

func withCell(f Front, h string) Front {
	f.CellCodehash = h
	return f
}

func withLayout(f Front, l string) Front {
	f.Layout = l
	return f
}

// On a builder chain (Ethereum, BNB Chain) the per-desk coverage rule runs, so each front names its quote contract's
// layout (only "builder") and its desk contracts' code hash: any other layout, none, or a missing, zero or short hash
// is refused at load. On a chain read at the head (Base) no quote contract is read, so neither applies: both must be
// unset, and are refused if set, with a message that says why.
func TestConfigLayoutAndCellCodehashByChain(t *testing.T) {
	f := ethFront("https://state.example/v1/1/state")
	require.NoError(t, (&Config{ChainID: valueobject.ChainIDEthereum, Fronts: []Front{f}}).validate())
	for name, bad := range map[string]Front{
		"no layout":      withLayout(f, ""),
		"another layout": withLayout(f, "storage"),
		"layout in caps": withLayout(f, "Builder"),
		"no codehash":    withCell(f, ""),
		"zero codehash":  withCell(f, "0x"+strings.Repeat("0", 64)),
		"short codehash": withCell(f, "0x4b9c"),
	} {
		cfg := Config{ChainID: valueobject.ChainIDEthereum, Fronts: []Front{bad}}
		err := cfg.validate()
		require.ErrorIs(t, err, ErrInvalidConfig, name)
		assert.Contains(t, err.Error(), "fronts[0]", name)
		_, perr := NewPoolsListUpdater(&cfg, nil)
		assert.ErrorIs(t, perr, ErrInvalidConfig, name)
		_, terr := NewPoolTracker(&cfg, nil)
		assert.ErrorIs(t, terr, ErrInvalidConfig, name)
		assert.ErrorIs(t, NewPoolFactory(&cfg, nil).cfgErr, ErrInvalidConfig, name)
	}
	base := testConfig().Fronts[0]
	require.NoError(t, (&Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{base}}).validate())
	for name, bad := range map[string]Front{
		"layout on Base":   withLayout(base, LayoutBuilder),
		"codehash on Base": withCell(base, f.CellCodehash),
	} {
		cfg := Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{bad}}
		err := cfg.validate()
		require.ErrorIs(t, err, ErrInvalidConfig, name)
		assert.Contains(t, err.Error(), "read at the head", name)
	}
}

// bnbFront is ethFront on BNB Chain: its /state read's timeout its chain's default, and bundleOnly set.
func bnbFront(urls ...string) Front {
	f := ethFront(urls...)
	f.Titan.Timeout, f.BundleOnly = 0, true
	return f
}

// BNB Chain's front is a builder build, read through /state as Ethereum's is: a front there names its layout, its desk
// code hash and its /state urls, and one read at the head (none of them) is refused. Its chain's tip floor is 0.1
// gwei, and its /state read runs on its 450 ms blocks: one fetch shared for 100 ms, a 100 ms merge window and a
// 400 ms timeout where the config sets none, and no configured timeout past one block. bundleOnly is BNB Chain's
// alone: off, the config loads and its listings are idle (TestBNBQuotesOnlyAFrontWhoseRoutesAreBundles); set on any
// other chain, it is refused.
func TestConfigOnBNBChain(t *testing.T) {
	cfg := Config{ChainID: valueobject.ChainIDBSC, Fronts: []Front{bnbFront("https://state.example/v1/56/state")}}
	require.NoError(t, cfg.validate())
	assert.True(t, cfg.clock().bundleOnly)
	for timeout, ok := range map[time.Duration]bool{450 * time.Millisecond: true, 451 * time.Millisecond: false,
		2 * time.Second: false} {
		c := Config{ChainID: valueobject.ChainIDBSC, Fronts: []Front{bnbFront("https://state.example/v1/56/state")}}
		c.Fronts[0].Titan.Timeout = timeout
		if ok {
			assert.NoError(t, c.validate(), "%s", timeout)
		} else {
			assert.ErrorIs(t, c.validate(), ErrInvalidConfig, "%s", timeout)
		}
	}
	off := Config{ChainID: valueobject.ChainIDBSC, Fronts: []Front{bnbFront("https://state.example/v1/56/state")}}
	off.Fronts[0].BundleOnly = false
	assert.NoError(t, off.validate(), "bundleOnly off loads")
	ethBundle := Config{ChainID: valueobject.ChainIDEthereum, Fronts: []Front{ethFront("https://s.example/v1/1/state")}}
	ethBundle.Fronts[0].BundleOnly = true
	assert.ErrorIs(t, ethBundle.validate(), ErrInvalidConfig, "bundleOnly on Ethereum")
	baseBundle := testConfig()
	baseBundle.Fronts[0].BundleOnly = true
	assert.ErrorIs(t, baseBundle.validate(), ErrInvalidConfig, "bundleOnly on Base")
	assert.True(t, cfg.builder())
	assert.True(t, cfg.Fronts[0].builder())
	assert.False(t, cfg.clock().beacon)
	assert.Equal(t, uint64(450), cfg.clock().blockMs)
	assert.True(t, cfg.clock().msPrecompile)
	assert.False(t, cfg.clock().classGas, "no class figure was measured on BNB Chain's build")
	assert.False(t, cfg.tipCapOK(99_999_999))
	assert.True(t, cfg.tipCapOK(100_000_000))
	assert.True(t, cfg.tipCapOK(0), "a cap of 0 is no cap on a builder build")

	tr, err := NewPoolTracker(&cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, 100*time.Millisecond, tr.stateEvery)
	f := cfg.Fronts[0]
	f.Titan.Timeout = 0
	assert.Equal(t, 400*time.Millisecond, tr.stateTimeout(&f))
	f.Titan.Timeout = 300 * time.Millisecond
	assert.Equal(t, 300*time.Millisecond, tr.stateTimeout(&f))

	eth := Config{ChainID: valueobject.ChainIDEthereum, Fronts: []Front{ethFront("https://state.example/v1/1/state")}}
	require.NoError(t, eth.validate())
	te, err := NewPoolTracker(&eth, nil)
	require.NoError(t, err)
	assert.Equal(t, stateShareWindow, te.stateEvery, "Ethereum's share window stands")
	f.Titan.Timeout = 0
	assert.Equal(t, defaultStateTimeout, te.stateTimeout(&f), "Ethereum's timeout stands")

	for name, fr := range map[string]Front{"read at the head": testConfig().Fronts[0],
		"no titan block": withTitan(ethFront(), nil)} {
		bad := Config{ChainID: valueobject.ChainIDBSC, Fronts: []Front{fr}}
		require.ErrorIs(t, bad.validate(), ErrInvalidConfig, name)
		_, terr := NewPoolTracker(&bad, nil)
		assert.ErrorIs(t, terr, ErrInvalidConfig, name)
		_, lerr := NewPoolsListUpdater(&bad, nil)
		assert.ErrorIs(t, lerr, ErrInvalidConfig, name)
	}
}

func withTitan(f Front, c *titan.Config) Front {
	f.Titan = c
	return f
}

// A listing's own gas is named once, by its id, with both figures measured, and never for a listing its front's
// credit listings name: on a builder chain and on any other.
func TestListingGasConfig(t *testing.T) {
	eth := func(credit []CreditListing, g ...ListingGas) Config {
		f := ethFront("https://state.example/v1/1/state")
		f.CreditListings, f.ListingGas = credit, g
		return Config{ChainID: valueobject.ChainIDEthereum, Fronts: []Front{f}}
	}
	own := ListingGas{Listing: 3, BaseIn: 145_430, QuoteIn: 136_333}
	good := eth(nil, own, ListingGas{Listing: 2, BaseIn: 144_744, QuoteIn: 138_338})
	require.NoError(t, good.validate())
	assert.Equal(t, &good.Fronts[0].ListingGas[0], good.Fronts[0].listingGas(3))
	assert.Nil(t, good.Fronts[0].listingGas(4))
	base := *testConfig()
	base.Fronts[0].ListingGas = []ListingGas{own}
	require.NoError(t, base.validate(), "on a chain read at the head too")

	zero, huge, noID, wide := own, own, own, own
	zero.QuoteIn, huge.BaseIn, noID.Listing, wide.Listing = 0, maxCreditGas+1, 0, 1<<24
	credit := []CreditListing{{Listing: 3, CustodyBaseIn: 1, CustodyQuoteIn: 1}}
	for name, bad := range map[string]Config{
		"a figure missing":  eth(nil, zero),
		"a figure too high": eth(nil, huge),
		"no listing":        eth(nil, noID),
		"past 24 bits":      eth(nil, wide),
		"named twice":       eth(nil, own, own),
		"a credit listing":  eth(credit, own),
	} {
		assert.ErrorIs(t, bad.validate(), ErrInvalidConfig, name)
	}
}
