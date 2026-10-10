package lotflow

// Typed form of the P2-FIX exact-transition fixtures (schema kyber-v3v4/P2-FIX/v1, described in the
// MemeFiLP repo at docs/kyber-v3v4/P2-FIX.md §4). A fixture is one exact-input swap on an in-fork
// NavJitHookV17 LOT/USDG venue with its complete semantic pre-state, outcome and per-candidate probe
// table. The simulator is validated against these; nothing here is used on the quoting path.

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"strings"
)

const (
	FixtureSchema = "kyber-v3v4/P2-FIX/v1"

	StatusOK     = "ok"
	StatusRevert = "revert"

	KindV4 = 1 // NavJitLegs.KIND_V4: a PoolManager pool
	KindV3 = 2 // NavJitLegs.KIND_V3: a Uniswap v3 pool

	VerdictFill   = "fill"
	VerdictNoFill = "nofill"

	ModeProbed = "probed"
)

// Big is a non-negative-or-signed integer that the fixtures encode as a decimal JSON string (or, for
// small values, a JSON number).
type Big struct{ big.Int }

func (b *Big) UnmarshalJSON(data []byte) error {
	s := string(bytes.Trim(data, `"`))
	if s == "null" || s == "" {
		return nil
	}
	base := 10
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		s, base = s[2:], 16
	}
	if _, ok := b.SetString(s, base); !ok {
		return fmt.Errorf("lotflow: bad integer %q", s)
	}
	return nil
}

// Copy returns a fresh *big.Int with b's value.
func (b *Big) Copy() *big.Int { return new(big.Int).Set(&b.Int) }

// Int is a small integer the fixtures encode either as a JSON number or as a decimal string.
type Int int64

func (n *Int) UnmarshalJSON(data []byte) error {
	s := string(bytes.Trim(data, `"`))
	if s == "null" || s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("lotflow: bad small integer %q: %w", s, err)
	}
	*n = Int(v)
	return nil
}

type FixtureIndex struct {
	Schema     string       `json:"schema"`
	Count      int          `json:"count"`
	Cases      []IndexEntry `json:"cases"`
	MultiCount int          `json:"multiCount"`
	MultiCases []MultiEntry `json:"multiCases"`
}

// MultiEntry indexes a multi-swap fixture (F-FIXTURES §3.1).
type MultiEntry struct {
	ID        string `json:"id"`
	File      string `json:"file"`
	SwapCount int    `json:"swapCount"`
	OneTx     bool   `json:"oneTx"`
}

type IndexEntry struct {
	ID                   string     `json:"id"`
	File                 string     `json:"file"`
	PinL2Block           int64      `json:"pinL2Block"`
	Lot                  string     `json:"lot"`
	Symbol               string     `json:"symbol"`
	Direction            string     `json:"direction"`
	Settlement           string     `json:"settlement"`
	AmountIn             Big        `json:"amountIn"`
	Status               string     `json:"status"`
	AmountOut            Big        `json:"amountOut"`
	TreasuryUsdgReceived Big        `json:"treasuryUsdgReceived"`
	Revert               string     `json:"revert"`
	Legs                 []IndexLeg `json:"legs"`
	Tags                 []string   `json:"tags"`
}

type IndexLeg struct {
	Stock          string `json:"stock"`
	Kind           int    `json:"kind"`
	CandidateIndex int    `json:"candidateIndex"`
	Buy            bool   `json:"buy"`
	Paid           Big    `json:"paid"`
	Got            Big    `json:"got"`
	TicksCrossed   int    `json:"ticksCrossed"`
}

// Case is one fixture (cases/<id>.json).
type Case struct {
	Schema  string                       `json:"schema"`
	ID      string                       `json:"id"`
	Note    string                       `json:"note"`
	Pin     Pin                          `json:"pin"`
	Input   Input                        `json:"input"`
	Pre     Pre                          `json:"pre"`
	Outcome Outcome                      `json:"outcome"`
	Post    *Post                        `json:"post"`
	Probes  []Probe                      `json:"probes"`
	Derived Derived                      `json:"derived"`
	Storage map[string]map[string]string `json:"storage"`
	Tags    []string                     `json:"tags"`
	// multi-swap shape: Swaps are complete single-swap fixtures in order; Tx is the one-transaction
	// route record when the swaps were also sent together.
	Shape     string    `json:"shape"`
	Swaps     []*Case   `json:"swaps"`
	SwapIndex Int       `json:"swapIndex"`
	Tx        *TxRecord `json:"tx"`
}

type TxRecord struct {
	Input struct {
		Legs []json.RawMessage `json:"legs"`
	} `json:"input"`
	Outcome struct {
		Status            string `json:"status"`
		LegOutputs        []Big  `json:"legOutputs"`
		MatchesSequential bool   `json:"matchesSequential"`
	} `json:"outcome"`
}

type Pin struct {
	ChainID        int64 `json:"chainId"`
	L2Block        int64 `json:"l2Block"`
	BlockTimestamp Big   `json:"blockTimestamp"`
	EvmBlockNumber Big   `json:"evmBlockNumber"`
}

type Input struct {
	Lot               string `json:"lot"`
	Direction         string `json:"direction"` // buyLot | sellLot
	TokenIn           string `json:"tokenIn"`
	TokenOut          string `json:"tokenOut"`
	AmountIn          Big    `json:"amountIn"`
	Kind              string `json:"kind"`
	ZeroForOne        bool   `json:"zeroForOne"`
	SqrtPriceLimitX96 Big    `json:"sqrtPriceLimitX96"`
	Settlement        string `json:"settlement"` // prefund | payAfter
	HookData          string `json:"hookData"`
	GasForwarded      Big    `json:"gasForwarded"`
	// A5: "venue" (default) | "plainV4" | "plainV3"; plain swaps name the constituent pool traded.
	Type     string          `json:"type"`
	Stock    string          `json:"stock"`
	SourceID string          `json:"sourceId"`
	A5       json.RawMessage `json:"a5"`
	A6       json.RawMessage `json:"a6"`
}

// IsBuy reports whether the swap pays USDG for LOT.
func (in *Input) IsBuy() bool { return in.Direction == "buyLot" }

type Pre struct {
	Hook         HookState     `json:"hook"`
	LegsConst    LegsConst     `json:"legsConst"`
	Venue        VenueState    `json:"venue"`
	Lot          LotState      `json:"lot"`
	PM           Balances      `json:"pm"`
	Nav          NavState      `json:"nav"`
	VenueBook    VenueBookInfo `json:"venueBook"`
	Constituents []Constituent `json:"constituents"`
	Mq           *MqPre        `json:"mq"`
}

// MqPre is pre.mq of an MQ fixture (F-FIXTURES §4).
type MqPre struct {
	Registry       string         `json:"registry"`
	Weth           string         `json:"weth"`
	ConversionPool string         `json:"conversionPool"`
	TwapWindow     int            `json:"twapWindow"`
	EthUsdFeed     MqFeedInfo     `json:"ethUsdFeed"`
	UsdgUsdFeed    MqFeedInfo     `json:"usdgUsdFeed"`
	SpotUsd18      map[string]Big `json:"spotUsd18"`
	Conversion     Candidate      `json:"conversion"`
	Twap           []MqTwapInfo   `json:"twap"`
}

type MqFeedInfo struct {
	Address       string `json:"address"`
	FeedDecimals  int    `json:"feedDecimals"`
	FeedAnswer    Big    `json:"feedAnswer"`
	FeedUpdatedAt Big    `json:"feedUpdatedAt"`
}

type MqTwapInfo struct {
	Pool                   string  `json:"pool"`
	Tick                   int     `json:"tick"`
	ObservationIndex       int     `json:"observationIndex"`
	ObservationCardinality int     `json:"observationCardinality"`
	TickCumulatives        []Big   `json:"tickCumulatives"`
	TwapTick               int     `json:"twapTick"`
	ObservationsRead       []MqObs `json:"observationsRead"`
}

type MqObs struct {
	Index          int  `json:"index"`
	BlockTimestamp Big  `json:"blockTimestamp"`
	TickCumulative Big  `json:"tickCumulative"`
	Initialized    bool `json:"initialized"`
}

// A6Drift is input.a6 of an MQ time-drift fixture.
type A6Drift struct {
	DtSeconds Int `json:"dtSeconds"`
	QuoteAtT  struct {
		Timestamp Big `json:"timestamp"`
		AmountOut Big `json:"amountOut"`
	} `json:"quoteAtT"`
	DriftBps Int `json:"driftBps"`
}

// A6Info parses input.a6 (nil when absent).
func (in *Input) A6Info() *A6Drift {
	if len(in.A6) == 0 || string(in.A6) == "null" {
		return nil
	}
	var d A6Drift
	if json.Unmarshal(in.A6, &d) != nil {
		return nil
	}
	return &d
}

type HookParams struct {
	BuySpreadBps  int64 `json:"buySpreadBps"`
	SellSpreadBps int64 `json:"sellSpreadBps"`
	BandBps       int64 `json:"bandBps"`
	SizeBufferBps int64 `json:"sizeBufferBps"`
	WidthTicks    int   `json:"widthTicks"`
}

type HookState struct {
	Address      string     `json:"address"`
	Params       HookParams `json:"params"`
	Treasury     string     `json:"treasury"`
	NavGuard     string     `json:"navGuard"`
	Usdg         string     `json:"usdg"`
	UsdgDecimals int        `json:"usdgDecimals"`
	VenueBook    string     `json:"venueBook"`
	V3Factory    string     `json:"v3Factory"`
	Registry     string     `json:"registry"`
	Owner        string     `json:"owner"`
	LotBalance   Big        `json:"lotBalance"`
	UsdgBalance  Big        `json:"usdgBalance"`
}

type LegsConst struct {
	ProbeGas       Big `json:"PROBE_GAS"`
	ProbeMargin    Big `json:"PROBE_MARGIN"`
	ProbeCallerGas Big `json:"PROBE_CALLER_GAS"`
	LegLimitBps    Big `json:"LEG_LIMIT_BPS"`
	MaxBookSources Big `json:"MAX_BOOK_SOURCES"` // pre-D-B sets
	MaxCandidates  Big `json:"MAX_CANDIDATES"`   // D-B sets
	ReadGas        Big `json:"READ_GAS"`
}

type PoolKey struct {
	Currency0   string `json:"currency0"`
	Currency1   string `json:"currency1"`
	Fee         uint32 `json:"fee"`
	TickSpacing int    `json:"tickSpacing"`
	Hooks       string `json:"hooks"`
}

type VenueState struct {
	Key             PoolKey `json:"key"`
	PoolID          string  `json:"poolId"`
	SqrtPriceX96    Big     `json:"sqrtPriceX96"`
	Tick            int     `json:"tick"`
	ProtocolFee     uint32  `json:"protocolFee"`
	LpFee           uint32  `json:"lpFee"`
	Liquidity       Big     `json:"liquidity"`
	UsdgIsCurrency0 bool    `json:"usdgIsCurrency0"`
	Standing        struct {
		Liquidity Big `json:"liquidity"`
		Lower     int `json:"lower"`
		Upper     int `json:"upper"`
	} `json:"standing"`
}

type LotState struct {
	Address     string `json:"address"`
	Symbol      string `json:"symbol"`
	Decimals    Int    `json:"decimals"`
	TotalSupply Big    `json:"totalSupply"`
	UnitBase    Big    `json:"unitBase"`
}

type Balances struct {
	LotBalance  Big `json:"lotBalance"`
	UsdgBalance Big `json:"usdgBalance"`
	WethBalance Big `json:"wethBalance"` // MQ sets
}

type NavParams struct {
	NavTolBps      int64 `json:"navTolBps"`
	StaleTolBps    int64 `json:"staleTolBps"`
	MaxFeedAge     int64 `json:"maxFeedAge"`
	MinCoverageBps int64 `json:"minCoverageBps"`
}

type NavState struct {
	CheckedNavPerUnit18 Big       `json:"checkedNavPerUnit18"`
	CheckedNavRevert    string    `json:"checkedNavRevert"` // set instead when NavGuard refused
	NavGuard            string    `json:"navGuard"`
	Params              NavParams `json:"params"`
}

type VenueBookInfo struct {
	Address           string `json:"address"`
	MaxExtra          int    `json:"maxExtra"`
	FamilyOfV3Factory int    `json:"familyOfV3Factory"`
}

type Constituent struct {
	Stock          string         `json:"stock"`
	Symbol         string         `json:"symbol"`
	Unit           Big            `json:"unit"`
	PmBalance      Big            `json:"pmBalance"`
	HookBalance    Big            `json:"hookBalance"`
	Nav            ConstituentNav `json:"nav"`
	Registry       RegistryInfo   `json:"registry"`
	Book           BookInfo       `json:"book"`
	Candidates     []Candidate    `json:"candidates"`
	CandidateCount Int            `json:"candidateCount"`
}

type ConstituentNav struct {
	SpotUsd18     Big    `json:"spotUsd18"`
	Feed          string `json:"feed"`
	FeedDecimals  int    `json:"feedDecimals"`
	FeedRoundID   Big    `json:"feedRoundId"`
	FeedAnswer    Big    `json:"feedAnswer"`
	FeedStartedAt Big    `json:"feedStartedAt"`
	FeedUpdatedAt Big    `json:"feedUpdatedAt"`
	OraclePaused  string `json:"oraclePaused"` // "true" | "false" | "unreadable"
}

type RegistryInfo struct {
	PriceIndex       int            `json:"priceIndex"`
	Stable           string         `json:"stable"`
	StockIsCurrency0 bool           `json:"stockIsCurrency0"`
	StockDecimals    int            `json:"stockDecimals"`
	StableDecimals   int            `json:"stableDecimals"`
	Enabled          bool           `json:"enabled"`
	StableIsUsdg     bool           `json:"stableIsUsdg"`
	IsWethToken      bool           `json:"isWethToken"`
	Pools            []RegistryPool `json:"pools"`
}

type RegistryPool struct {
	Key          PoolKey `json:"key"`
	PoolID       string  `json:"poolId"`
	SqrtPriceX96 Big     `json:"sqrtPriceX96"`
	Liquidity    Big     `json:"liquidity"`
	Admission    string  `json:"admission"`
}

type BookInfo struct {
	VenuesOf string      `json:"venuesOf"` // "ok" | "reverted"
	Venues   []BookVenue `json:"venues"`
}

// BookVenue is one VenueBook entry with the three admission reads. A read that failed under READ_GAS
// (or returned < 32 bytes) is the string "unreadable"; ReadOK reports it.
type BookVenue struct {
	Pool             string          `json:"pool"`
	Family           int             `json:"family"`
	ReadFactory      string          `json:"readFactory"`
	ReadSqrtPriceX96 json.RawMessage `json:"readSqrtPriceX96"`
	ReadLiquidity    json.RawMessage `json:"readLiquidity"`
	ReadTickSpacing  json.RawMessage `json:"readTickSpacing"`
	Admission        string          `json:"admission"`
}

type LegLimit struct {
	Buy        Big `json:"buy"`
	Sell       Big `json:"sell"`
	ZeroForOne Big `json:"zeroForOne"`
	OneForZero Big `json:"oneForZero"`
}

type TickWindow struct {
	TickAtLimitZeroForOne int `json:"tickAtLimitZeroForOne"`
	TickAtLimitOneForZero int `json:"tickAtLimitOneForZero"`
	WordLo                int `json:"wordLo"`
	WordHi                int `json:"wordHi"`
	KeptTickLo            int `json:"keptTickLo"`
	KeptTickHi            int `json:"keptTickHi"`
}

type BitmapWord struct {
	Word   int `json:"word"`
	Bitmap Big `json:"bitmap"`
}

type TickInfo struct {
	Tick           int `json:"tick"`
	LiquidityGross Big `json:"liquidityGross"`
	LiquidityNet   Big `json:"liquidityNet"`
}

// Candidate is one admitted source in NavJitLegs.sources() order.
type Candidate struct {
	Kind     int    `json:"kind"`
	KindName string `json:"kindName"`
	SourceID string `json:"sourceId"`
	StockIs0 bool   `json:"stockIs0"`
	// v4
	Key         *PoolKey `json:"key"`
	ProtocolFee uint32   `json:"protocolFee"`
	LpFee       uint32   `json:"lpFee"`
	DynamicFee  bool     `json:"dynamicFee"`
	// v3
	Pool        string `json:"pool"`
	Family      int    `json:"family"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	Factory     string `json:"factory"`
	Fee         uint32 `json:"fee"`
	FeeProtocol uint32 `json:"feeProtocol"`
	// state
	SqrtPriceX96 Big          `json:"sqrtPriceX96"`
	Tick         int          `json:"tick"`
	Liquidity    Big          `json:"liquidity"`
	TickSpacing  int          `json:"tickSpacing"`
	LegLimit     LegLimit     `json:"legLimit"`
	TickWindow   TickWindow   `json:"tickWindow"`
	Words        []BitmapWord `json:"words"`
	Ticks        []TickInfo   `json:"ticks"`
	TickCount    Int          `json:"tickCount"`
}

type Outcome struct {
	Status               string `json:"status"`
	AmountOut            Big    `json:"amountOut"`
	TraderInSpent        Big    `json:"traderInSpent"`
	TraderOutReceived    Big    `json:"traderOutReceived"`
	TreasuryUsdgReceived Big    `json:"treasuryUsdgReceived"`
	GasUsedRouterCall    Big    `json:"gasUsedRouterCall"`
	RevertData           string `json:"revertData"`
	Hash                 string `json:"hash"`
	// standing cases: what the hook owner received, and the standing position after (also on reverts)
	Owner *struct {
		UsdgDelta Big `json:"usdgDelta"`
		LotDelta  Big `json:"lotDelta"`
	} `json:"owner"`
	StandingAfter *struct {
		Liquidity Big `json:"liquidity"`
		Lower     int `json:"lower"`
		Upper     int `json:"upper"`
	} `json:"standingAfter"`
}

type Post struct {
	Venue struct {
		SqrtPriceX96 Big `json:"sqrtPriceX96"`
		Tick         int `json:"tick"`
		Liquidity    Big `json:"liquidity"`
	} `json:"venue"`
	PM   Balances `json:"pm"`
	Hook Balances `json:"hook"`
	Lot  struct {
		TotalSupply Big `json:"totalSupply"`
	} `json:"lot"`
}

// Probe is one executed leg's probe table (P2-FIX §4.7): every candidate re-measured at the pre-state.
type Probe struct {
	Stock            string           `json:"stock"`
	Buy              bool             `json:"buy"`
	ConstituentIndex Int              `json:"constituentIndex"`
	Amount           Big              `json:"amount"`
	Mode             string           `json:"mode"`
	Candidates       []ProbeCandidate `json:"candidates"`
}

type ProbeCandidate struct {
	SourceID       string     `json:"sourceId"`
	Verdict        string     `json:"verdict"`
	ProbeResult    *Big       `json:"probeResult"`
	Exec           *ProbeExec `json:"exec"`
	ExecutedInSwap bool       `json:"executedInSwap"`
}

type ProbeExec struct {
	OK               bool `json:"ok"`
	Result           *Big `json:"result"`
	LegGas           Big  `json:"legGas"`
	ObservedSwapFee  *Int `json:"observedSwapFee"`
	PostSqrtPriceX96 *Big `json:"postSqrtPriceX96"`
	PostTick         *Int `json:"postTick"`
	PostLiquidity    *Big `json:"postLiquidity"`
}

type Derived struct {
	Revert               *DecodedError     `json:"revert"`
	RevertInnermost      *DecodedError     `json:"revertInnermost"`
	Legs                 []DerivedLeg      `json:"legs"`
	VenueSwaps           []VenueSwap       `json:"venueSwaps"`
	VenueModifyLiquidity []ModifyLiquidity `json:"venueModifyLiquidity"`
	BasketMinted         []Big             `json:"basketMinted"`
	BasketRedeemed       []Big             `json:"basketRedeemed"`
}

type DecodedError struct {
	Selector string   `json:"selector"`
	Name     string   `json:"name"`
	Args     []string `json:"args"`
	// Wrapped is set on a v4 WrappedError(target, hookSelector, reason, details).
	Wrapped *WrappedRevert `json:"wrapped"`
}

type WrappedRevert struct {
	Target       string        `json:"target"`
	HookSelector string        `json:"hookSelector"` // 0x575e24b4 beforeSwap / 0xb47b2fb1 afterSwap
	Reason       *DecodedError `json:"reason"`
}

type DerivedLeg struct {
	LogIndex         int    `json:"logIndex"`
	Stock            string `json:"stock"`
	ConstituentIndex int    `json:"constituentIndex"`
	CandidateIndex   int    `json:"candidateIndex"`
	Kind             int    `json:"kind"`
	Source           string `json:"source"`
	Buy              bool   `json:"buy"`
	Paid             Big    `json:"paid"`
	Got              Big    `json:"got"`
	PreTick          int    `json:"preTick"`
	SwapFeePips      int64  `json:"swapFeePips"`
}

type VenueSwap struct {
	PoolID       string `json:"poolId"`
	Sender       string `json:"sender"`
	Amount0      Big    `json:"amount0"`
	Amount1      Big    `json:"amount1"`
	SqrtPriceX96 Big    `json:"sqrtPriceX96"`
	Liquidity    Big    `json:"liquidity"`
	Tick         int    `json:"tick"`
	Fee          int64  `json:"fee"`
	LogIndex     int    `json:"logIndex"`
}

type ModifyLiquidity struct {
	PoolID         string `json:"poolId"`
	TickLower      int    `json:"tickLower"`
	TickUpper      int    `json:"tickUpper"`
	LiquidityDelta Big    `json:"liquidityDelta"`
	LogIndex       int    `json:"logIndex"`
}

// IsFill reports whether the fixture's swap succeeded.
func (c *Case) IsFill() bool { return c.Outcome.Status == StatusOK }

func LoadIndex(path string) (*FixtureIndex, error) {
	var idx FixtureIndex
	if err := readJSON(path, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

func LoadCase(path string) (*Case, error) {
	var c Case
	if err := readJSON(path, &c); err != nil {
		return nil, err
	}
	if c.Schema != FixtureSchema {
		return nil, fmt.Errorf("lotflow: %s: schema %q, want %q", path, c.Schema, FixtureSchema)
	}
	return &c, nil
}

// readFile reads path, or path+".gz" gunzipped when only that exists (the fixtures are stored gzipped).
func readFile(path string) ([]byte, error) {
	if !strings.HasSuffix(path, ".gz") {
		b, err := os.ReadFile(path)
		if !errors.Is(err, os.ErrNotExist) {
			return b, err
		}
		if _, gzErr := os.Stat(path + ".gz"); gzErr != nil {
			return nil, err
		}
		path += ".gz"
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("lotflow: %s: %w", path, err)
	}
	return io.ReadAll(zr)
}

func readJSON(path string, v any) error {
	b, err := readFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("lotflow: %s: %w", path, err)
	}
	return nil
}
