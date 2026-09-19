package spireprop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type rawSide struct {
	SpreadBps int16
	DepthBps  uint16
	Filled    *big.Int
	KnotCount uint8
}
type rawPair struct {
	Seq          uint64
	FillSeq      uint64
	LastUpdateAt uint64
	Mid          *big.Int
	QUnit        *big.Int
	Ask          rawSide
	Bid          rawSide
}
type rawKnot struct {
	Q     *big.Int
	Extra *big.Int
}
type PoolTracker struct{ client *ethrpc.Client }

var _ = pooltrack.RegisterFactoryE0(DexType, NewPoolTracker)

func NewPoolTracker(client *ethrpc.Client) *PoolTracker { return &PoolTracker{client} }

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	if len(p.Tokens) != 2 || p.Tokens[0] == nil || p.Tokens[1] == nil {
		return p, ErrInvalidState
	}
	var static StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &static); err != nil {
		return p, err
	}
	for _, addr := range []string{static.Entrypoint, static.CurveBook, static.Custodian, p.Tokens[0].Address, p.Tokens[1].Address} {
		if !validAddress(addr) {
			return p, ErrInvalidState
		}
	}
	header, err := t.client.GetETHClient().HeaderByNumber(ctx, nil)
	if err != nil {
		return p, err
	}
	base, quote := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)
	var wrapped struct{ V rawPair }
	var extra Extra
	var cUnit *big.Int
	var reserves [2]*big.Int
	var curveEntry, curveQuote, entryCurve, entryCustody, entryQuote, custEntry, custCurve common.Address
	req := t.client.NewRequest().SetContext(ctx).SetBlockHash(header.Hash())
	req.AddCall(&ethrpc.Call{ABI: curveBookABI, Target: static.CurveBook, Method: "pair", Params: []any{base}}, []any{&wrapped})
	req.AddCall(&ethrpc.Call{ABI: curveBookABI, Target: static.CurveBook, Method: "validUntil", Params: []any{base}}, []any{&extra.ValidUntil})
	req.AddCall(&ethrpc.Call{ABI: curveBookABI, Target: static.CurveBook, Method: "ttl"}, []any{&extra.TTL})
	req.AddCall(&ethrpc.Call{ABI: curveBookABI, Target: static.CurveBook, Method: "cUnit"}, []any{&cUnit})
	for i, token := range []common.Address{base, quote} {
		req.AddCall(&ethrpc.Call{ABI: custodianABI, Target: static.Custodian, Method: "availableLiquidity", Params: []any{token}}, []any{&reserves[i]})
	}
	req.AddCall(&ethrpc.Call{ABI: curveBookABI, Target: static.CurveBook, Method: "entrypoint"}, []any{&curveEntry})
	req.AddCall(&ethrpc.Call{ABI: curveBookABI, Target: static.CurveBook, Method: "quoteToken"}, []any{&curveQuote})
	req.AddCall(&ethrpc.Call{ABI: entrypointABI, Target: static.Entrypoint, Method: "curveBook"}, []any{&entryCurve})
	req.AddCall(&ethrpc.Call{ABI: entrypointABI, Target: static.Entrypoint, Method: "custodian"}, []any{&entryCustody})
	req.AddCall(&ethrpc.Call{ABI: entrypointABI, Target: static.Entrypoint, Method: "quoteToken"}, []any{&entryQuote})
	req.AddCall(&ethrpc.Call{ABI: custodianABI, Target: static.Custodian, Method: "entrypoint"}, []any{&custEntry})
	req.AddCall(&ethrpc.Call{ABI: custodianABI, Target: static.Custodian, Method: "curveBook"}, []any{&custCurve})
	result, err := req.Aggregate()
	if err != nil {
		return p, err
	}
	if result.BlockNumber == nil || result.BlockNumber.Cmp(header.Number) != 0 {
		return p, ErrInvalidState
	}
	if curveEntry != common.HexToAddress(static.Entrypoint) || curveQuote != quote || entryCurve != common.HexToAddress(static.CurveBook) || entryCustody != common.HexToAddress(static.Custodian) || entryQuote != quote || custEntry != curveEntry || custCurve != entryCurve {
		return p, ErrInvalidState
	}
	raw := wrapped.V
	extra.Seq, extra.FillSeq, extra.LastUpdateAt, extra.BlockTimestamp = raw.Seq, raw.FillSeq, raw.LastUpdateAt, header.Time
	inputs := []*big.Int{raw.Mid, raw.QUnit, cUnit, raw.Ask.Filled, raw.Bid.Filled}
	outputs := []*uint256.Int{&extra.Mid, &extra.QUnit, &extra.CUnit, &extra.Ask.Filled, &extra.Bid.Filled}
	for i, value := range inputs {
		if value == nil || value.Sign() < 0 || outputs[i].SetFromBig(value) {
			return p, ErrInvalidState
		}
	}
	for _, reserve := range reserves {
		if reserve == nil || reserve.Sign() < 0 || reserve.BitLen() > 256 {
			return p, ErrInvalidState
		}
	}
	extra.Ask.SpreadBps, extra.Ask.DepthBps = raw.Ask.SpreadBps, raw.Ask.DepthBps
	extra.Bid.SpreadBps, extra.Bid.DepthBps = raw.Bid.SpreadBps, raw.Bid.DepthBps
	if raw.Ask.KnotCount > 36 || raw.Bid.KnotCount > 36 {
		return p, ErrInvalidState
	}
	knots := [2][]rawKnot{make([]rawKnot, int(raw.Ask.KnotCount)), make([]rawKnot, int(raw.Bid.KnotCount))}
	req = t.client.NewRequest().SetContext(ctx).SetBlockHash(header.Hash())
	for side := range knots {
		for i := range knots[side] {
			k := &knots[side][i]
			req.AddCall(&ethrpc.Call{ABI: curveBookABI, Target: static.CurveBook, Method: "knot", Params: []any{base, uint8(side), big.NewInt(int64(i))}}, []any{k})
		}
	}
	if len(req.Calls) > 0 {
		if _, err := req.Aggregate(); err != nil {
			return p, err
		}
	}
	for i, side := range []*Side{&extra.Ask, &extra.Bid} {
		side.Knots = make([]Knot, len(knots[i]))
		for j, k := range knots[i] {
			if k.Q == nil || k.Extra == nil || k.Q.Sign() < 0 || k.Extra.Sign() < 0 || side.Knots[j].Q.SetFromBig(k.Q) || side.Knots[j].Extra.SetFromBig(k.Extra) {
				return p, ErrInvalidState
			}
		}
	}
	// Hash-pinned calls remain internally coherent through a reorg; reject an
	// orphaned snapshot before exposing it to routing.
	canonical, err := t.client.GetETHClient().HeaderByNumber(ctx, header.Number)
	if err != nil {
		return p, err
	}
	if canonical.Hash() != header.Hash() {
		return p, ErrInvalidState
	}
	if err := (&PoolSimulator{Extra: extra}).validate(); err != nil {
		return p, err
	}
	encoded, err := json.Marshal(&extra)
	if err != nil {
		return p, err
	}
	p.Extra, p.BlockNumber, p.Timestamp = string(encoded), header.Number.Uint64(), time.Now().Unix()
	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	return p, nil
}
